package handlers

import (
	"hongik-backend/service"
)

type Handler struct {
	store       service.Store
	interpreter *service.InterpreterService
	cache       *service.Cache
}

func New(store service.Store, interpreter *service.InterpreterService, cache *service.Cache) *Handler {
	return &Handler{
		store:       store,
		interpreter: interpreter,
		cache:       cache,
	}
}
