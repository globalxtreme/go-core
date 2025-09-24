package privateapi

import (
	"errors"
	xtremeapi "github.com/globalxtreme/go-core/v2/api"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"os"
)

type BusinessWorkflowAPI interface {
	NotificationPush(payload interface{}) xtremeres.ResponseSuccessWithPagination
}

func NewBusinessWorkflowAPI() (BusinessWorkflowAPI, error) {
	host := os.Getenv("CLIENT_PRIVATE_API_BUSINESS_WORKFLOW_HOST")
	clientId := os.Getenv("CLIENT_PRIVATE_API_BUSINESS_WORKFLOW_ID")
	clientName := os.Getenv("CLIENT_PRIVATE_API_BUSINESS_WORKFLOW_NAME")
	clientSecret := os.Getenv("CLIENT_PRIVATE_API_BUSINESS_WORKFLOW_SECRET")

	if host == "" || clientId == "" || clientName == "" || clientSecret == "" {
		return nil, errors.New("Please set private api Business Workflow environment variables")
	}

	client := xtremeapi.NewXtremeAPI(xtremeapi.XtremeAPIOption{
		Headers: map[string]string{
			"Client-ID":     clientId,
			"Client-Name":   clientName,
			"Client-Secret": clientSecret,
		},
	})

	api := businessWorkflowAPI{
		baseURL: host,
		client:  client,
	}

	return &api, nil
}

type businessWorkflowAPI struct {
	baseURL string
	client  xtremeapi.XtremeAPI
}

func (api *businessWorkflowAPI) NotificationPush(payload interface{}) xtremeres.ResponseSuccessWithPagination {
	return api.client.Post(api.baseURL+"/notifications", payload)
}
