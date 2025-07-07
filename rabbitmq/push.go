package xtremerabbitmq

import (
	"context"
	"encoding/json"
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	"github.com/rabbitmq/amqp091-go"
	"log"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type RabbitMQ struct {
	Connection string
	Exchange   string
	Queue      string
	Deliveries []rabbitMQDelivery
	Data       interface{}
	MessageId  *int
	SenderId   *string
	SenderType *string
	Timeout    *time.Duration

	service    string
	body       interface{}
	properties publishingProperties
}

type rabbitMQDelivery struct {
	Service          string
	NeedNotification bool
}

func (mq *RabbitMQ) OnConnection(connection string) *RabbitMQ {
	mq.Connection = connection

	return mq
}

func (mq *RabbitMQ) OnExchange(exchange string) *RabbitMQ {
	mq.Exchange = exchange

	return mq
}

func (mq *RabbitMQ) OnQueue(queue string) *RabbitMQ {
	mq.Queue = queue

	return mq
}

func (mq *RabbitMQ) OnDelivery(service string, needNotificationArg ...bool) *RabbitMQ {
	delivery := rabbitMQDelivery{
		Service: service,
	}

	if len(needNotificationArg) > 0 {
		delivery.NeedNotification = needNotificationArg[0]
	}

	mq.Deliveries = append(mq.Deliveries, delivery)

	return mq
}

func (mq *RabbitMQ) OnSender(senderId any, senderType string) *RabbitMQ {
	var strSenderId string
	switch senderId.(type) {
	case string:
		strSenderId = senderId.(string)
	case uint:
		strSenderId = strconv.Itoa(int(senderId.(uint)))
	case int:
		strSenderId = strconv.Itoa(senderId.(int))
	}

	mq.SenderId = &strSenderId
	mq.SenderType = &senderType

	return mq
}

func (mq *RabbitMQ) WithTimeout(duration time.Duration) *RabbitMQ {
	mq.Timeout = &duration

	return mq
}

func (mq *RabbitMQ) Push() {
	mq.service = os.Getenv("SERVICE")

	mq.setupMessage()
	mq.publishMessage()
}

func (mq *RabbitMQ) setupMessage() *RabbitMQ {
	mqConnection, ok := RabbitMQConnectionCache[mq.Connection]
	if !ok {
		if len(RabbitMQConnectionCache) == 0 {
			RabbitMQConnectionCache = make(map[string]xtrememodel.RabbitMQConnection)
		}

		mqConnQuery := RabbitMQSQL.Where("connection = ?", mq.Connection)
		if mq.Connection == RABBITMQ_CONNECTION_LOCAL {
			mqConnQuery = mqConnQuery.Where("service = ?", mq.service)
		}

		err := mqConnQuery.First(&mqConnection).Error
		if err != nil || mqConnection.ID == 0 {
			log.Panicf("Data connection does not exists: %s", err)
		}

		RabbitMQConnectionCache[mq.Connection] = mqConnection
	}

	var message xtrememodel.RabbitMQMessage
	if mq.MessageId != nil {
		RabbitMQSQL.First(&message, mq.MessageId)
	}

	correlationId, _ := exec.Command("uuidgen").Output()
	mq.properties = publishingProperties{
		CorrelationId: string(correlationId),
		DeliveryMode:  amqp091.Persistent,
		ContentType:   "application/json",
	}

	payload := map[string]interface{}{
		"data":      mq.Data,
		"messageId": mq.MessageId,
	}

	if message.ID == 0 {
		message.ConnectionId = mqConnection.ID
		message.Exchange = mq.Exchange
		message.Queue = mq.Queue
		message.SenderId = mq.SenderId
		message.SenderType = mq.SenderType
		message.SenderService = mq.service
		message.Payload = payload

		err := RabbitMQSQL.Create(&message).Error
		if err == nil {
			payload["messageId"] = message.ID

			message.Payload = payload
			RabbitMQSQL.Save(&message)

			if mq.Deliveries != nil && len(mq.Deliveries) > 0 {
				msgDeliveries := make([]xtrememodel.RabbitMQMessageDelivery, 0)
				for _, delivery := range mq.Deliveries {
					msgDeliveries = append(msgDeliveries, xtrememodel.RabbitMQMessageDelivery{
						MessageId:        message.ID,
						ConsumerService:  delivery.Service,
						NeedNotification: delivery.NeedNotification,
						StatusId:         RABBITMQ_MESSAGE_DELIVERY_STATUS_PENDING_ID,
					})
				}

				RabbitMQSQL.Create(&msgDeliveries)
			}
		} else {
			log.Panicf("Unable to save message: %s", err)
		}
	}

	mq.body = payload
	return mq
}

func (mq *RabbitMQ) publishMessage() {
	conn, ok := RabbitMQConnectionDial[mq.Connection]
	if !ok {
		log.Panicf("Please init rabbitmq connection first")
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Panicf("Failed to open a channel: %s", err)
	}
	defer ch.Close()

	var timeout time.Duration
	if mq.Timeout == nil {
		timeout = 10 * time.Second
	} else {
		timeout = *mq.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	body, _ := json.Marshal(mq.body)

	if mq.Exchange != "" {
		err = ch.ExchangeDeclare(
			mq.Exchange,
			"fanout",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Panicf("Failed to declare a exchange: %s", err)
		}

		err = ch.PublishWithContext(ctx,
			mq.Exchange,
			"",
			false,
			false,
			amqp091.Publishing{
				CorrelationId: mq.properties.CorrelationId,
				DeliveryMode:  mq.properties.DeliveryMode,
				ContentType:   mq.properties.ContentType,
				Body:          body,
			})
		if err != nil {
			log.Panicf("Failed to publish a message: %s", err)
		}
	} else if mq.Queue != "" {
		q, err := ch.QueueDeclare(
			mq.Queue,
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Panicf("Failed to declare a queue: %s", err)
		}

		err = ch.PublishWithContext(ctx,
			"",
			q.Name,
			false,
			false,
			amqp091.Publishing{
				CorrelationId: mq.properties.CorrelationId,
				DeliveryMode:  mq.properties.DeliveryMode,
				ContentType:   mq.properties.ContentType,
				Body:          body,
			})
		if err != nil {
			log.Panicf("Failed to send a message: %s", err)
		}
	}
}
