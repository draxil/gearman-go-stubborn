package stubborn

import (
	"github.com/mikespook/gearman-go/worker"
	"time"
)


/*

Create a stubborn worker.ErrorHandler for gearman-go, which will attempt 
to maintain a peristant connection to the server.
 
Can be as simple as:
 
 w.ErrorHandler = stubborn.MakeErrorHandler( nil );

*/
func MakeErrorHandler( settings *Settings )( worker.ErrorHandler ){
	if( settings == nil ){
		settings = &Settings{ ReconnectDelay : 250  * time.Millisecond }
	}

	if( settings.ReconnectDelay == 0 ){
		// never reconnect immediately, that's insane.
		settings.ReconnectDelay = 250 * time.Millisecond;
	}

	return func ( e error ){

		wdc, wdcok := e.(*worker.WorkerDisconnectError)

		if( wdcok ){
			if settings.ShouldReconnectHandler == nil ||
			   settings.ShouldReconnectHandler( wdc ) {
				go func(){
					var rc_err error
					not_tried_yet := true
					for( (rc_err != nil || not_tried_yet) && 
						(settings.ShouldReconnectHandler == nil ||
						settings.ShouldReconnectHandler( wdc )) ){
						time.Sleep( settings.ReconnectDelay )
						rc_err = wdc.Reconnect()
						not_tried_yet = false;
					}
					if( rc_err != nil && settings.ErrorHandler != nil ){
						settings.ErrorHandler( rc_err )
					}
				}()
			}
		} else {
			if( settings.ErrorHandler != nil ){
				settings.ErrorHandler( e )
			}
		}
	};
}

type ShouldReconnectHandler func(*worker.WorkerDisconnectError)(bool);

/* 
 Settings for the stubbon worker error handler. 
 Currently allows you to provide:
 
 ErrorHandler to actually report on non D/C errors
 ShouldReconnectHandler to influence whether or not to reconnect
 ReconnectDelay to influence how long to wait. Won't accept 0! I'd advise at least (250 * time.Millisecond)

 */
type Settings struct{
	ErrorHandler  worker.ErrorHandler
	ShouldReconnectHandler ShouldReconnectHandler
	ReconnectDelay time.Duration
}


/* 
 Shortcut to create a gearman-go worker with a stubborn error handler pre-installed
 
 limit is the limit value passed to worker.New
 Your worker will have already had an w.ErrorHandler installed.
*/
func NewStubbornWorker( limit int, settings * Settings )(* worker.Worker){
	w := worker.New( limit );
	w.ErrorHandler = MakeErrorHandler( settings )
	return w;
}