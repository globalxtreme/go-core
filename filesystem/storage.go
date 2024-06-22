package xtremefs

import (
	"github.com/globalxtreme/go-core/v2/pkg"
	"github.com/gorilla/mux"
	"net/http"
	"os"
)

type Storage struct {
	IsPublic bool
}

func (repo Storage) GetFullPath(path string) string {
	baseDir, _ := os.Getwd()

	var storageDir string
	if repo.IsPublic {
		storageDir = xtremepkg.SetStorageAppPublicDir(path)
	} else {
		storageDir = xtremepkg.SetStorageDir(path)
	}

	return baseDir + "/" + storageDir
}

func (repo Storage) GetFullPathURL(path string) string {
	return os.Getenv("API_GATEWAY_LINK_URL") + path
}

func (repo Storage) ShowFile(w http.ResponseWriter, r *http.Request, paths ...string) {
	var path string

	if len(paths) > 0 {
		path = paths[0]
	} else {
		vars := mux.Vars(r)
		path = vars["path"]
	}

	if repo.IsPublic {
		path = xtremepkg.SetStorageAppPublicDir(path)
	} else {
		path = xtremepkg.SetStorageDir(path)
	}

	realPath := storageCheckPath(path)
	if realPath == nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, realPath.(string))
}
