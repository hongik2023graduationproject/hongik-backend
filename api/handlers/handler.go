package handlers

import (
	"hongik-backend/service"
)

type Handler struct {
	store       service.Store
	interpreter *service.InterpreterService
}

func New(store service.Store, interpreter *service.InterpreterService) *Handler {
	return &Handler{
		store:       store,
		interpreter: interpreter,
	}
}
