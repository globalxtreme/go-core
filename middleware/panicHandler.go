package xtrememdw

import (
	"encoding/json"
	"fmt"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	"github.com/globalxtreme/go-core/v2/response"
	"log"
	"net/http"
	"os"
)

func PanicHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.Header().Set("Content-Type", "application/json")

				var res *xtremeres.ResponseError
				if panicData, ok := r.(*xtremeres.ResponseError); ok {
					res = panicData
				} else {
					res = &xtremeres.ResponseError{
						Status: xtremeres.StatusError{
							Bug: true,
							Status: xtremeres.Status{
								Code:    http.StatusInternalServerError,
								Message: "An error Occurred.",
							},
						},
					}
				}

				fmt.Fprintf(os.Stderr, "panic: %v\n", r)
				xtremepkg.LogError(r, res.Status.Bug)

				w.WriteHeader(res.Status.Code)

				jsonData, err := json.Marshal(res)
				if err != nil {
					log.Println("Failed to marshal error response:", err)
					return
				}

				w.Write(jsonData)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
