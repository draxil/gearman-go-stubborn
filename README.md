gearman-go-stubborn
===================

Stubborn helpers for gearman-go

Currently, a default error handler for https://github.com/mikespook/gearman-go which will maintain server connections on disconnect, stubbonly trying to reconnect to servers until they come back.

[![Build Status](https://travis-ci.org/draxil/gearman-go-stubborn.png?branch=master)](https://travis-ci.org/draxil/gearman-go-stubborn)
[![GoDoc](https://godoc.org/github.com/draxil/gearman-go-stubborn/worker/stubborn?status.png)](https://godoc.org/github.com/draxil/gearman-go-stubborn/worker/stubborn)

Basic usage:
  
  
  import (
    "github.com/mikespook/gearman-go/worker"
    "github.com/draxil/gearman-go-stubborn/worker/stubborn"
  )
  
  // then:
  w := worker.New(worker.Unlimited)
  w.ErrorHandler = MakeErrorHandler( nil )
  
Or for the lazy:

  import "github.com/draxil/gearman-go-stubborn/worker/stubborn"
  
  w := NewStubbornWorker(worker.Unlimited, nil)
  
Which will invoke the creation of the worker and assign the error handler for you. You can pass in settings to influence the behaviour of the handler and/or layer on your own error handler for non D/C errors. 

See the godoc for full details (or the tests if you are brave).
