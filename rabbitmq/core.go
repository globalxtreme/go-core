package xtremerabbitmq

import (
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	"github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
	"time"
)

const RABBITMQ_CONNECTION_GLOBAL = "global"
const RABBITMQ_CONNECTION_LOCAL = "local"

const RABBITMQ_MESSAGE_DELIVERY_STATUS_PENDING_ID = 1
const RABBITMQ_MESSAGE_DELIVERY_STATUS_PENDING = "Pending"
const RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH_ID = 2
const RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH = "Finish"
const RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID = 3
const RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR = "Error"

const RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING_ID = 1
const RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING = "Pending"
const RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID = 2
const RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING = "Processing"
const RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID = 3
const RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS = "Success"
const RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID = 4
const RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR = "Error"

var (
	RabbitMQSQL  *gorm.DB
	RabbitMQConf rabbitmqconf

	RabbitMQConnectionDial  map[string]*amqp091.Connection
	RabbitMQConnectionCache map[string]xtrememodel.RabbitMQConnection
)

type rabbitmqconf struct {
	Queue      string
	Connection map[string]RabbitMQConnectionConf
	Exchange   RabbitMQExchangeConf
	Timeout    time.Duration
}

type RabbitMQConnectionConf struct {
	Host     string
	Port     string
	Username string
	Password string
}

type RabbitMQExchangeConf struct {
	Name       string
	Type       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp091.Table
}

type RabbitMQDeliveryResponse struct {
	Status rabbitMQDeliveryResponseStatus `json:"status"`
	Error  rabbitMQDeliveryResponseError  `json:"error"`
	Result interface{}                    `json:"result"`
}

type rabbitMQDeliveryResponseStatus struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type rabbitMQDeliveryResponseError struct {
	Message string `json:"message"`
	Trace   string `json:"trace"`
}

type AsyncWorkflowForm struct {
	WorkflowId uint `json:"workflowId"`
	StepOrder  int  `json:"stepOrder"`
}

type publishingProperties struct {
	CorrelationId string
	DeliveryMode  uint8
	ContentType   string
}

type RabbitMQMessageDeliveryStatus struct{}

func (cons RabbitMQMessageDeliveryStatus) OptionIDNames() map[int]string {
	return map[int]string{
		RABBITMQ_MESSAGE_DELIVERY_STATUS_PENDING_ID: RABBITMQ_MESSAGE_DELIVERY_STATUS_PENDING,
		RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH_ID:  RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH,
		RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID:   RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR,
	}
}

func (cons RabbitMQMessageDeliveryStatus) IDAndName(id int) map[string]interface{} {
	return map[string]interface{}{
		"id":   id,
		"name": cons.Display(id),
	}
}

func (cons RabbitMQMessageDeliveryStatus) Display(id int) string {
	idNames := cons.OptionIDNames()
	if name, ok := idNames[id]; ok {
		return name
	}
	return ""
}

type RabbitMQAsyncWorkflowStatus struct{}

func (cons RabbitMQAsyncWorkflowStatus) OptionIDNames() map[int]string {
	return map[int]string{
		RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING_ID:    RABBITMQ_ASYNC_WORKFLOW_STATUS_PENDING,
		RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING_ID: RABBITMQ_ASYNC_WORKFLOW_STATUS_PROCESSING,
		RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS_ID:    RABBITMQ_ASYNC_WORKFLOW_STATUS_SUCCESS,
		RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR_ID:      RABBITMQ_ASYNC_WORKFLOW_STATUS_ERROR,
	}
}

func (cons RabbitMQAsyncWorkflowStatus) IDAndName(id int) map[string]interface{} {
	return map[string]interface{}{
		"id":   id,
		"name": cons.Display(id),
	}
}

func (cons RabbitMQAsyncWorkflowStatus) Display(id int) string {
	idNames := cons.OptionIDNames()
	if name, ok := idNames[id]; ok {
		return name
	}
	return ""
}
