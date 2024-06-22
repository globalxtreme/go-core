package handler

import (
	xtremecore "github.com/globalxtreme/go-core"
	"github.com/gorilla/mux"
	"net/http"
)

type BaseStorageHandler struct{}

func (ctr BaseStorageHandler) ShowFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	storage := xtremecore.Storage{IsPublic: true}
	storage.ShowFile(w, r, vars["path"])
}
