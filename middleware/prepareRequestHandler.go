package middleware

import (
	"github.com/globalxtreme/go-core/response"
	"net/http"
	"strings"
)

func PrepareRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		if strings.Contains(contentType, "multipart/form-data") {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				response.ErrXtremePayloadVeryLarge("")
			}
		} else if contentType == "application/json" || contentType == "application/x-www-form-urlencoded" {
			err := r.ParseForm()
			if err != nil {
				response.ErrXtremeBadRequest("Unable to parse form!")
			}
		}
		next.ServeHTTP(w, r)
	})
}
