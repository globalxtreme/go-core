package handler

import (
	"github.com/globalxtreme/gobaseconf/filesystem"
	"github.com/gorilla/mux"
	"net/http"
)

type BaseStorageHandler struct{}

func (ctr BaseStorageHandler) ShowFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	storage := filesystem.Storage{IsPublic: true}
	storage.ShowFile(w, r, vars["path"])
}
