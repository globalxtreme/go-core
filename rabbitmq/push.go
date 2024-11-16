package xtremerabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	"github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

var (
	RabbitMQSQL  *gorm.DB
	RabbitMQConf rabbitmqconf
)

type RabbitMQ struct {
	Data       interface{}
	Queues     []string
	Type       string
	Name       string
	RouteKey   string
	Key        string
	MessageId  *int
	Body       interface{}
	Properties publishingProperties
	SenderId   *uint
	SenderType *string
}

type publishingProperties struct {
	CorrelationId string
	DeliveryMode  uint8
	ContentType   string
}

func (mq *RabbitMQ) OnQueue(queue string) *RabbitMQ {
	mq.Queues = append(mq.Queues, queue)

	return mq
}

func (mq *RabbitMQ) OnSender(sId uint, sType string) *RabbitMQ {
	mq.SenderId = &sId
	mq.SenderType = &sType

	return mq
}

func (mq *RabbitMQ) SetType(sType string) *RabbitMQ {
	mq.Type = sType

	return mq
}

func (mq *RabbitMQ) SetName(sName string) *RabbitMQ {
	mq.Name = sName

	return mq
}

func (mq *RabbitMQ) SetRouteKey(sRouteKey string) *RabbitMQ {
	mq.RouteKey = sRouteKey

	return mq
}

func (mq *RabbitMQ) Push() {
	mq.setupMessage()
	mq.publishMessage()
}

func (mq *RabbitMQ) setupMessage() *RabbitMQ {
	config := RabbitMQConf

	var message xtrememodel.RabbitMQMessage

	if mq.MessageId != nil {
		RabbitMQSQL.First(&message, mq.MessageId)
	}

	correlationId, _ := exec.Command("uuidgen").Output()

	msgContent := map[string]interface{}{
		"key":       mq.Key,
		"exchange":  mq.Name,
		"queue":     config.Queue,
		"message":   mq.Data,
		"messageId": mq.MessageId,
	}

	mq.Properties = publishingProperties{
		CorrelationId: string(correlationId),
		DeliveryMode:  amqp091.Persistent,
		ContentType:   "application/json",
	}

	if message.ID == 0 {
		message.Statuses = make(xtrememodel.MapBoolColumn)
		for _, queue := range mq.Queues {
			message.Statuses[queue] = false
		}

		payload := map[string]interface{}{
			"body":       msgContent,
			"properties": mq.Properties,
		}

		message.Exchange = mq.Name
		message.QueueSender = config.Queue
		message.QueueConsumers = mq.Queues
		message.Key = mq.Key
		message.SenderId = mq.SenderId
		message.SenderType = mq.SenderType
		message.Payload = payload

		err := RabbitMQSQL.Create(&message).Error
		if err == nil {
			msgContent["messageId"] = message.ID
			payload["body"] = msgContent

			message.Payload = payload
			RabbitMQSQL.Save(&message)
		} else {
			log.Panicf("Unable to save message: %s", err)
		}
	}

	mq.Body = msgContent
	return mq
}

func (mq *RabbitMQ) publishMessage() {
	config := RabbitMQConf
	connConf := config.Connection
	exchange := config.Exchange

	conn, err := amqp091.Dial(fmt.Sprintf("amqp://%s:%s@%s:%s/", connConf.Username, connConf.Password, connConf.Host, connConf.Port))
	if err != nil {
		log.Panicf("Failed to connect to RabbitMQ: %s", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	err = ch.ExchangeDeclare(
		mq.Name,
		mq.Type,
		exchange.Durable,
		exchange.AutoDelete,
		exchange.Internal,
		exchange.NoWait,
		exchange.Args,
	)
	if err != nil {
		log.Panicf("Failed to declare a exchange: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	body, _ := json.Marshal(mq.Body)

	for _, queue := range mq.Queues {
		routingKey := ""
		if mq.RouteKey != "" {
			routingKey = queue
		}

		err = ch.PublishWithContext(ctx,
			mq.Name,
			routingKey,
			false,
			false,
			amqp091.Publishing{
				CorrelationId: mq.Properties.CorrelationId,
				DeliveryMode:  mq.Properties.DeliveryMode,
				ContentType:   mq.Properties.ContentType,
				Body:          body,
			})

		if err != nil {
			log.Panicf("Failed to publish a message: %s", err)
		}
	}
}
