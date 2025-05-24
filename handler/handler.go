package handler

import (
	"fmt"
	xtremefs "github.com/globalxtreme/go-core/v2/filesystem"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"github.com/gorilla/mux"
	"net/http"
)

type Handler struct{}

func (Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func (Handler) StorageShowFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	storage := xtremefs.Storage{IsPublic: true}
	storage.ShowFile(w, r, vars["path"])
}

func (Handler) LogActivate(w http.ResponseWriter, r *http.Request) {
	xtremepkg.LOG_ACTIVE = !xtremepkg.LOG_ACTIVE
	status := "inactive"
	if xtremepkg.LOG_ACTIVE {
		status = "active"
	}

	res := xtremeres.Response{Object: map[string]interface{}{"log": status}}

	res.Success(w)
}
