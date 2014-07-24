package stubborn;
/*

 Stubborn error handler for "github.com/mikespook/gearman-go/worker".

 Responds to connection errors by attempting to re-connect!
 
 Designed to be a useful default error handler if you want a gearman worker which stubbonly 
 maintains it's server connections. You can influence this bull headed behaviour with callbacks.



*/
