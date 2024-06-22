package handler

import (
	xtremepkg "github.com/globalxtreme/go-core/pkg"
	"github.com/gorilla/mux"
	"net/http"
)

type BaseStorageHandler struct{}

func (ctr BaseStorageHandler) ShowFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	storage := xtremepkg.Storage{IsPublic: true}
	storage.ShowFile(w, r, vars["path"])
}
