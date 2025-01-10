package handler

import (
	"net/http"

	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
)

type BaseLogHandler struct{}

func (ctr BaseLogHandler) Activate(w http.ResponseWriter, r *http.Request) {
	xtremepkg.LOG_ACTIVE = !xtremepkg.LOG_ACTIVE
	status := "inactive"
	if xtremepkg.LOG_ACTIVE {
		status = "active"
	}

	res := xtremeres.Response{Object: map[string]interface{}{"log": status}}

	res.Success(w)
}
