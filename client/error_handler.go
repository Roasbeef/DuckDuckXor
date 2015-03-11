package main

import "fmt"

func (e *errorHandler) Error(errMessage error, quit chan struct{}) {
	select {
	case <-quit:
		return
	case e.Message <- errMessage:
	}
}

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
