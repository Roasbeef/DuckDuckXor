package main

import "fmt"

type errorHandler struct {
	Message chan error
}

func NewErrorHandler() *errorHandler {
	e := make(chan error, 1)
	return &errorHandler{Message: e}
}

func (e *errorHandler) Start() {
	go e.handleErrors()

}
func (e *errorHandler) handleErrors() {
	for {
		select {
		case err := <-e.Message:
			fmt.Printf(err.Error())
		}

	}
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
