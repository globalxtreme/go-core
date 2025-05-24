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

	h := handler.Handler{}
	router.HandleFunc("/health-check", h.HealthCheck).Methods("GET")
	router.HandleFunc("/storages/{path:.*}", h.StorageShowFile).Methods("GET")
	router.HandleFunc("/{path:.*}/log-active", h.LogActivate).Methods("POST")

	callback(router)
}
