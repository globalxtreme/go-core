package xtremeres

import (
	"net/http"
)

func Error(code int, message string, internalMsg string, attributes any) {
	panic(&ResponseError{
		Status: Status{
			Code:        code,
			Message:     message,
			InternalMsg: internalMsg,
			Attributes:  attributes,
		},
	})
}

func ErrXtremeUnauthenticated(internalMsg string) {
	Error(http.StatusUnauthorized, "Unauthenticated.", internalMsg, nil)
}

func ErrXtremeBadRequest(internalMsg string) {
	Error(http.StatusBadRequest, "Bad request!", internalMsg, nil)
}

func ErrXtremePayloadVeryLarge(internalMsg string) {
	Error(http.StatusRequestEntityTooLarge, "Your payload very large!", internalMsg, nil)
}

func ErrXtremeValidation(attributes []interface{}) {
	Error(http.StatusBadRequest, "Missing Required Parameter", "", attributes)
}

func ErrXtremeNotFound(internalMsg string) {
	Error(http.StatusNotFound, "Data not found", internalMsg, nil)
}

func ErrXtremeUploadFile(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to upload file", internalMsg, nil)
}
