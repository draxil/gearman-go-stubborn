package stubborn

import (
	"errors"
	"github.com/mikespook/gearman-go/client"
	"github.com/mikespook/gearman-go/worker"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

const port = `3700`
const gearmand = `/usr/sbin/gearmand`

var gearman_ready chan bool
var kill_gearman chan bool

func init() {

	if check_gearman_present() {
		panic(`Something already listening on our testing port. Chickening out of testing with it!`)
	}
	gearman_ready = make(chan bool)
	kill_gearman = make(chan bool)
}

func run_gearman() {
	gm_cmd := exec.Command(gearmand, `--port`, port)
	start_err := gm_cmd.Start()

	if start_err != nil {
		panic(`could not start gearman, aborting test :` + start_err.Error())
	}

	// Make sure we clear up our gearman:
	defer func() {
		gm_cmd.Process.Kill()
	}()

	for tries := 10; tries > 0; tries-- {
		if check_gearman_present() {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	if !check_gearman_present() {
		panic(`Unable to start gearman aborting test`)
	}
	gearman_ready <- true

	<-kill_gearman
}

func check_gearman_present() bool {
	con, err := net.Dial(`tcp`, `127.0.0.1:`+port)
	if err != nil {
		return false
	}
	con.Close()
	return true
}

func check_gearman_is_dead() bool {

	for tries := 10; tries > 0; tries-- {
		if !check_gearman_present() {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
func skip_if_no_gearman(t *testing.T) {
	_, err := os.Stat(gearmand)
	if err != nil && os.IsNotExist(err) {
		t.Skip("Skipping no real gearman to test with")
	}
}
func TestBasicDisconnectWithStubbornErrh(t *testing.T) {
	skip_if_no_gearman(t)
	go run_gearman()
	<-gearman_ready

	w := worker.New(worker.Unlimited)
	timeout := make(chan bool, 1)
	done := make(chan bool, 1)

	w.ErrorHandler = MakeErrorHandler(nil)

	if err := w.AddFunc("stubborn-job",
		func(j worker.Job) (b []byte, e error) {
			log.Println("job happens")

			done <- true
			return
		}, 0); err != nil {
		t.Error(err)
	}
	if err := w.AddServer(worker.Network, "127.0.0.1:"+port); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	err := w.Ready()

	if err != nil {
		t.Error(err)
	}

	go w.Work()
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}
	// First check worker is in place:
	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: initial job")
	}

	kill_gearman <- true

	check_gearman_is_dead()
	//
	go run_gearman()
	<-gearman_ready
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}

	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: reconnect job")
	}

	w.Close()
	kill_gearman <- true
}

func TestBasicDisconnectWithStubbornWorker(t *testing.T) {
	skip_if_no_gearman(t)
	check_gearman_is_dead()
	go run_gearman()
	<-gearman_ready

	w := NewStubbornWorker(worker.Unlimited, nil)
	timeout := make(chan bool, 1)
	done := make(chan bool, 1)

	if err := w.AddFunc("stubborn-job",
		func(j worker.Job) (b []byte, e error) {
			log.Println("job happens")

			done <- true
			return
		}, 0); err != nil {
		t.Error(err)
	}
	if err := w.AddServer(worker.Network, "127.0.0.1:"+port); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	err := w.Ready()

	if err != nil {
		t.Error(err)
	}

	go w.Work()
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}
	// First check worker is in place:
	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: initial job")
	}

	kill_gearman <- true

	check_gearman_is_dead()
	//
	go run_gearman()
	<-gearman_ready
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}

	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: reconnect job")
	}

	w.Close()
	kill_gearman <- true
}

func TestBasicDisconnectWithShouldReconnectHandler(t *testing.T) {
	skip_if_no_gearman(t)
	check_gearman_is_dead()
	go run_gearman()
	<-gearman_ready

	timeout := make(chan bool, 1)
	done := make(chan bool, 1)

	should_reconnect := true
	var should_reconnect_mutex sync.Mutex
	srh := func(*worker.WorkerDisconnectError) bool {
		defer func() {
			should_reconnect_mutex.Unlock()
		}()
		should_reconnect_mutex.Lock()
		return should_reconnect
	}

	w := NewStubbornWorker(worker.Unlimited, &Settings{
		ShouldReconnectHandler: srh,
		ReconnectDelay:         10 * time.Millisecond, // make this SUPER quick for this one
	})

	if err := w.AddFunc("stubborn-job",
		func(j worker.Job) (b []byte, e error) {
			log.Println("job happens")

			done <- true
			return
		}, 0); err != nil {
		t.Error(err)
	}
	if err := w.AddServer(worker.Network, "127.0.0.1:"+port); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	err := w.Ready()

	if err != nil {
		t.Error(err)
	}

	go w.Work()
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}
	// First check worker is in place:
	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: initial job")
	}

	kill_gearman <- true

	check_gearman_is_dead()
	//
	go run_gearman()
	<-gearman_ready
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}

	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: reconnect job")
	}

	quick_timeout_test := func() bool {
		kill_gearman <- true
		check_gearman_is_dead()
		//
		go run_gearman()
		<-gearman_ready
		if !check_gearman_present() {
			t.Error("Sanity: no gearman server?")
		}

		send_client_request()
		quick_timeout := make(chan bool, 1)
		go func() {
			time.Sleep(20 * time.Millisecond)
			quick_timeout <- true
		}()

		select {
		case <-done:
			return false
		case <-quick_timeout:
			return true
		}

	}

	if quick_timeout_test() {
		t.Error("Control test for cancelling reconnect failed. Dubious! Or the test isn't locked down well enough")
	}

	// NOTE: to sanity-check this test you can set this to true and check we DO reconnect faster than quick_timeout,
	// we could even do that first for completeness?
	should_reconnect_mutex.Lock()
	should_reconnect = false
	should_reconnect_mutex.Unlock()

	if !quick_timeout_test() {
		t.Error("Seemingly the ShouldReconnectHandler didn't stop us reconnecting!")
	}

	w.Close()
	kill_gearman <- true
}
func TestBasicDisconnectWithSubErrHandler(t *testing.T) {
	skip_if_no_gearman(t)
	check_gearman_is_dead()
	go run_gearman()
	<-gearman_ready

	timeout := make(chan bool, 1)
	done := make(chan bool, 1)
	errdone := make(chan bool, 1)

	handled := false
	var handled_mutex sync.Mutex
	erh := func(e error) {
		handled_mutex.Lock()
		handled = true
		handled_mutex.Unlock()
		log.Println(e)
		errdone <- true
	}
	was_handled := func() (rv bool) {
		handled_mutex.Lock()
		rv = handled
		handled_mutex.Unlock()
		return
	}

	w := NewStubbornWorker(worker.Unlimited, &Settings{
		ShouldReconnectHandler: func(*worker.WorkerDisconnectError) bool { log.Println("SHOULD RC?"); return false },
		ErrorHandler:           erh,
	})
	job := 0
	if err := w.AddFunc("stubborn-job",
		func(j worker.Job) (b []byte, e error) {
			log.Println("job happens")

			if job == 1 {
				e = errors.New("woo")
				return
			}
			job++

			done <- true
			return
		}, 0); err != nil {
		t.Error(err)
	}
	if err := w.AddServer(worker.Network, "127.0.0.1:"+port); err != nil {
		t.Error(err)
	}

	go func() {
		time.Sleep(5 * time.Second)
		timeout <- true
	}()

	err := w.Ready()

	if err != nil {
		t.Error(err)
	}

	go w.Work()
	if !check_gearman_present() {
		t.Error("Sanity: no gearman server?")
	}
	// First check worker is in place:
	send_client_request()

	select {
	case <-done:
	case <-timeout:
		t.Error("Test timed out: initial job")
	case <-errdone:
		t.Error("Unexpected error?")
	}

	if was_handled() {
		t.Error("unexpected fire of error handler")
	}
	send_client_request()

	select {
	case <-errdone:
	case <-timeout:
		t.Error("Test timed out: err")
	}

	if !was_handled() {
		t.Error("Seemingly errh did not fire?")
	}

	kill_gearman <- true

	w.Close()
}
func send_client_request() {
	c, err := client.New(worker.Network, "127.0.0.1:"+port)
	if err == nil {
		_, err = c.Do("stubborn-job", []byte{}, client.JobHigh, nil)
		if err != nil {
			log.Println("error sending client request " + err.Error())
		}

	} else {
		log.Println("error with client " + err.Error())
	}
}
