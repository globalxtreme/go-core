package middleware

import (
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"net/http"
	"os"
	"strings"
)

func PrepareRequestHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")

		if strings.Contains(contentType, "multipart/form-data") {
			maxPayload := 32
			if maxPayloadENV := os.Getenv("MAX_PAYLOAD"); maxPayloadENV != "" {
				maxPayload = xtremepkg.ToInt(maxPayloadENV)
			}

			err := r.ParseMultipartForm(int64(maxPayload << 20))
			if err != nil {
				xtremeres.ErrXtremePayloadVeryLarge("")
			}
		} else if contentType == "application/json" || contentType == "application/x-www-form-urlencoded" {
			err := r.ParseForm()
			if err != nil {
				xtremeres.ErrXtremeBadRequest("Unable to parse form!")
			}
		}
		next.ServeHTTP(w, r)
	})
}
