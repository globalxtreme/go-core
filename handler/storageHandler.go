package handler

import (
	xtremefs "github.com/globalxtreme/go-core/v2/filesystem"
	"github.com/gorilla/mux"
	"net/http"
)

type BaseStorageHandler struct{}

func (ctr BaseStorageHandler) ShowFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	storage := xtremefs.Storage{IsPublic: true}
	storage.ShowFile(w, r, vars["path"])
}
