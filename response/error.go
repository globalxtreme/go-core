package xtremeres

import (
	"net/http"
)

func Error(code int, message string, internalMsg string, bug bool, attributes any) {
	panic(&ResponseError{
		Status: StatusError{
			Bug: bug,
			Status: Status{
				Code:        code,
				Message:     message,
				InternalMsg: internalMsg,
				Attributes:  attributes,
			},
		},
	})
}

func ErrXtremeUnauthenticated(internalMsg string) {
	Error(http.StatusUnauthorized, "Unauthenticated.", internalMsg, false, nil)
}

func ErrXtremeBadRequest(internalMsg string) {
	Error(http.StatusBadRequest, "Bad request!", internalMsg, false, nil)
}

func ErrXtremePayloadVeryLarge(internalMsg string) {
	Error(http.StatusRequestEntityTooLarge, "Your payload very large!", internalMsg, false, nil)
}

func ErrXtremeValidation(attributes []interface{}) {
	Error(http.StatusBadRequest, "Missing Required Parameter", "", false, attributes)
}

func ErrXtremeNotFound(internalMsg string) {
	Error(http.StatusNotFound, "Data not found", internalMsg, false, nil)
}

func ErrXtremeUploadFile(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to upload file", internalMsg, false, nil)
}

func ErrXtremeDeleteFile(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to delete file", internalMsg, false, nil)
}

func ErrXtremeUUID(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to generate uuid", internalMsg, false, nil)
}

func ErrXtremeAPI(internalMsg string) {
	Error(http.StatusInternalServerError, "Calling external api is invalid!", internalMsg, false, nil)
}

func ErrXtremeRabbitMQMessageGet(internalMsg string) {
	Error(http.StatusNotFound, "RabbitMQ message not found", internalMsg, false, nil)
}

func ErrXtremeRabbitMQMessageDeliveryGet(internalMsg string) {
	Error(http.StatusNotFound, "RabbitMQ message delivery not found", internalMsg, false, nil)
}

func ErrXtremeRabbitMQMessageDeliveryValidation(internalMsg string) {
	Error(http.StatusBadRequest, "RabbitMQ message delivery form invalid", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowGet(internalMsg string) {
	Error(http.StatusNotFound, "RabbitMQ async workflow not found", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowSave(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to create rabbitMQ async workflow", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowValidation(internalMsg string) {
	Error(http.StatusBadRequest, "RabbitMQ async workflow form invalid", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowStepGet(internalMsg string) {
	Error(http.StatusNotFound, "RabbitMQ async workflow step not found", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowStepSave(internalMsg string) {
	Error(http.StatusInternalServerError, "Unable to create rabbitMQ async workflow step", internalMsg, false, nil)
}

func ErrXtremeRabbitMQAsyncWorkflowStepValidation(internalMsg string) {
	Error(http.StatusBadRequest, "RabbitMQ async workflow step form invalid", internalMsg, false, nil)
}
