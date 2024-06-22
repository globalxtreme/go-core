package xtremefs

import (
	"encoding/base64"
	"github.com/gabriel-vasile/mimetype"
	"github.com/globalxtreme/go-core/pkg"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
		st.Name = xtremepkg.RandomString(20)
	}

	file, handler, err := r.FormFile(param)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var storagePath string
	if st.IsPublic {
		storagePath = xtremepkg.SetStorageAppPublicDir()
	} else {
		storagePath = xtremepkg.SetStorageAppDir()
	}

	xtremepkg.CheckAndCreateDirectory(storagePath + "/" + st.Path)

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
		st.Name = xtremepkg.RandomString(20)
	}

	var storagePath string
	if st.IsPublic {
		storagePath = xtremepkg.SetStorageAppPublicDir()
	} else {
		storagePath = xtremepkg.SetStorageAppDir()
	}

	xtremepkg.CheckAndCreateDirectory(storagePath + "/" + st.Path)

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
