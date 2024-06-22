package xtremecore

import (
	"encoding/base64"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gorilla/mux"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Storage struct {
	IsPublic bool
}

func (repo Storage) GetFullPath(path string) string {
	baseDir, _ := os.Getwd()

	var storageDir string
	if repo.IsPublic {
		storageDir = SetStorageAppPublicDir(path)
	} else {
		storageDir = SetStorageDir(path)
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
		path = SetStorageAppPublicDir(path)
	} else {
		path = SetStorageDir(path)
	}

	realPath := storageCheckPath(path)
	if realPath == nil {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, realPath.(string))
}

type Uploader struct {
	Path     string
	Name     string
	IsPublic bool
}

func (st Uploader) SetPath(path string) Uploader {
	st.Path = path

	return st
}

func (st Uploader) SetName(name string) Uploader {
	st.Name = name

	return st
}

func (st Uploader) MoveFile(r *http.Request, param string) (any, error) {
	if len(st.Name) == 0 {
		st.Name = RandomString(20)
	}

	file, handler, err := r.FormFile(param)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var storagePath string
	if st.IsPublic {
		storagePath = SetStorageAppPublicDir()
	} else {
		storagePath = SetStorageAppDir()
	}

	CheckAndCreateDirectory(storagePath + "/" + st.Path)

	filename := st.Name + filepath.Ext(handler.Filename)

	destinationFile, err := os.Create(strings.Replace(storagePath+"/"+st.Path+"/"+filename, "//", "/", -1))
	if err != nil {
		return nil, err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, file)
	if err != nil {
		return nil, err
	}

	return strings.Replace(st.Path+"/"+filename, "//", "/", -1), nil
}

func (st Uploader) MoveContent(content string) (any, error) {
	if len(st.Name) == 0 {
		st.Name = RandomString(20)
	}

	var storagePath string
	if st.IsPublic {
		storagePath = SetStorageAppPublicDir()
	} else {
		storagePath = SetStorageAppDir()
	}

	CheckAndCreateDirectory(storagePath + "/" + st.Path)

	fileBytes, err := base64.StdEncoding.DecodeString(content)
	if err != nil {
		return nil, err
	}

	mime := mimetype.Detect(fileBytes)
	st.Name = st.Name + mime.Extension()

	err = ioutil.WriteFile(strings.Replace(storagePath+"/"+st.Path+"/"+st.Name, "//", "/", -1), fileBytes, 0777)
	if err != nil {
		return nil, err
	}

	return strings.Replace(st.Path+"/"+st.Name, "//", "/", -1), nil
}

func storageCheckPath(path string) any {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
	}

	if info.IsDir() {
		return nil
	}

	return path
}
