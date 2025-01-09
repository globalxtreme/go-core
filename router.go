package xtremecore

import (
	"github.com/globalxtreme/go-core/v2/handler"
	xtrememdw "github.com/globalxtreme/go-core/v2/middleware"
	"github.com/gorilla/mux"
)

type CallbackRouter func(*mux.Router)

func RegisterRouter(router *mux.Router, callback CallbackRouter) {
	router.Use(xtrememdw.PanicHandler)
	router.Use(xtrememdw.PrepareRequestHandler)

	// Storage route
	stHandler := handler.BaseStorageHandler{}
	router.HandleFunc("/storages/{path:.*}", stHandler.ShowFile).Methods("GET")

	// Log route
	logHandler := handler.BaseLogHandler{}
	router.HandleFunc("/log-active", logHandler.Activate).Methods("POST")

	callback(router)
}
