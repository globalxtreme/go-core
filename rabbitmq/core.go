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
const RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH = "Pending"
const RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID = 3
const RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR = "Pending"

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
