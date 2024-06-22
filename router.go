package xtremecore

import (
	"github.com/globalxtreme/go-core/v2/handler"
	"github.com/globalxtreme/go-core/v2/middleware"
	"github.com/gorilla/mux"
)

type CallbackRouter func(*mux.Router)

func RegisterRouter(router *mux.Router, callback CallbackRouter) {
	router.Use(middleware.PanicHandler)
	router.Use(middleware.PrepareRequestHandler)

	// Storage route
	stHandler := handler.BaseStorageHandler{}
	router.HandleFunc("/storages/{path:.*}", stHandler.ShowFile).Methods("GET")

	callback(router)
}
