package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
)

type errorHandler struct {
	Message   chan error
	stopFuncs []func() error
	stopChan  chan func() error
}

func NewErrorHandler() *errorHandler {
	e := make(chan error, 1)

	return &errorHandler{
		Message:   e,
		stopChan:  make(chan func() error),
		stopFuncs: make([]func() error, 10),
	}
}

func (e *errorHandler) Start() {
	go e.handleErrors()
	go e.handleOSCalls()
	go e.collectStopFuncs()

}
func (e *errorHandler) handleErrors() {
	for {
		select {
		case err := <-e.Message:
			fmt.Printf(err.Error())
			for _, stop := range e.stopFuncs {
				stop()
			}
		}

	}
}

func (e *errorHandler) collectStopFuncs() {
	for {
		select {
		case a := <-e.stopChan:
			e.stopFuncs = append(e.stopFuncs, a)
		}

	}

}

func (e *errorHandler) handleOSCalls() {
	c := make(chan os.Signal, 1)

	signal.Notify(c, os.Interrupt, os.Kill)
	fmt.Printf("waiting for message\n")
	s := <-c
	fmt.Printf("got message!")
	e.Message <- errors.New(s.String())

}

func (e *errorHandler) createAbortFunc() func(chan struct{}, error) {
	return func(quitChan chan struct{}, err error) {
	out:
		for {
			select {
			case <-quitChan:
				break out
			case e.Message <- err:
				break out
			}
		}

	}
}
