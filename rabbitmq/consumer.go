package xtremerabbitmq

import (
	"encoding/json"
	"fmt"
	xtrememodel "github.com/globalxtreme/go-core/v2/model"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremeres "github.com/globalxtreme/go-core/v2/response"
	"github.com/rabbitmq/amqp091-go"
	"log"
	"strings"
	"time"
)

type RabbitMQConsumerInterface interface {
	Consume(message xtrememodel.RabbitMQMessage) (interface{}, error)
}

type RabbitMQConsumeOpt struct {
	Exchange string
	Queue    string
	Consumer RabbitMQConsumerInterface
}

type rabbitMQBody struct {
	MessageId uint `json:"messageId"`
	Data      any  `json:"data"`
}

func Consume(connection string, options []RabbitMQConsumeOpt) {
	if connection == "" || (connection != RABBITMQ_CONNECTION_GLOBAL && connection != RABBITMQ_CONNECTION_LOCAL) {
		log.Panicf("Please choose connection %s or %s", RABBITMQ_CONNECTION_GLOBAL, RABBITMQ_CONNECTION_LOCAL)
	}

	for _, opt := range options {
		if (opt.Exchange == "" && opt.Queue == "") || (opt.Exchange != "" && opt.Queue != "") {
			log.Panicf("Please select one of them: Exhange or Queue!!")
		}
	}

	mqConnection, ok := RabbitMQConnectionCache[connection]
	if !ok {
		if len(RabbitMQConnectionCache) == 0 {
			RabbitMQConnectionCache = make(map[string]xtrememodel.RabbitMQConnection)
		}

		mqConnQuery := RabbitMQSQL.Where("connection = ?", connection)
		if connection == RABBITMQ_CONNECTION_LOCAL {
			mqConnQuery = mqConnQuery.Where("service = ?", xtremepkg.GetServiceName())
		}

		err := mqConnQuery.First(&mqConnection).Error
		if err != nil || mqConnection.ID == 0 {
			log.Panicf("Data connection does not exists: %s", err)
		}

		RabbitMQConnectionCache[connection] = mqConnection
	}

	connConf := RabbitMQConf.Connection[connection]
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

	var forever chan struct{}

	for _, opt := range options {
		if opt.Exchange != "" {
			fanoutConsumer(ch, mqConnection, opt)
		} else if opt.Queue != "" {
			directConsumer(ch, mqConnection, opt)
		}
	}

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

func PrepareManualConsumer(form AsyncTransactionForm, response *map[string]interface{}) (pushNotification func()) {
	var messageDelivery struct {
		MessageId        uint                                 `gorm:"column:messageId"`
		DeliveryId       uint                                 `gorm:"column:deliveryId"`
		Connection       string                               `gorm:"column:connection"`
		Exchange         string                               `gorm:"column:exchange"`
		Queue            string                               `gorm:"column:queue"`
		SenderId         string                               `gorm:"column:senderId"`
		SenderType       string                               `gorm:"column:senderType"`
		SenderService    string                               `gorm:"column:senderService"`
		StatusId         int                                  `gorm:"column:statusId"`
		NeedNotification bool                                 `gorm:"column:needNotification"`
		Responses        *xtrememodel.ArrayMapInterfaceColumn `gorm:"column:responses"`
	}

	if form.MessageId > 0 && form.SenderId != "" {
		err := RabbitMQSQL.Table("message_deliveries as deliveries").
			Select(
				"messages.id as messageId",
				"deliveries.id as deliveryId",
				"connections.connection",
				"messages.exchange",
				"messages.queue",
				"messages.senderId",
				"messages.senderType",
				"messages.senderService",
				"deliveries.statusId",
				"deliveries.needNotification",
				"deliveries.responses",
			).
			Joins(`INNER JOIN messages on deliveries.messageId = messages.id`).
			Joins(`INNER JOIN connections on messages.connectionId = connections.id`).
			Where(`messages.id = ? AND messages.senderId = ? AND deliveries.consumerService = ?`,
				form.MessageId, form.SenderId, xtremepkg.GetServiceName()).
			Scan(&messageDelivery).Error
		if err != nil || messageDelivery.DeliveryId == 0 {
			xtremeres.ErrXtremeRabbitMQMessageDeliveryValidation("")
		}

		if messageDelivery.StatusId != RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID {
			xtremeres.ErrXtremeRabbitMQMessageDeliveryValidation("Status message delivery is not error!")
		}
	}

	pushNotification = func() {
		if messageDelivery.DeliveryId > 0 {
			if r := recover(); r != nil {
				panic(r)
			}

			deliveryResponses := make([]map[string]interface{}, 0)
			if messageDelivery.Responses != nil {
				deliveryResponses = *messageDelivery.Responses
			}

			if response != nil {
				deliveryResponses = append(deliveryResponses, *response)
			}

			finishId := RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH_ID
			RabbitMQSQL.Model(&xtrememodel.RabbitMQMessageDelivery{}).
				Where(`id = ?`, messageDelivery.DeliveryId).
				Updates(&xtrememodel.RabbitMQMessageDelivery{
					StatusId:  finishId,
					Responses: (*xtrememodel.ArrayMapInterfaceColumn)(&deliveryResponses),
				})

			if messageDelivery.NeedNotification {
				queue := ""
				if messageDelivery.Exchange != "" {
					queue = setQueueKey(messageDelivery.Exchange)
				} else if messageDelivery.Queue != "" {
					queue = setQueueKey(messageDelivery.Queue)
				}

				if queue != "" {
					deliveryRes := RabbitMQDeliveryResponse{
						Status: rabbitMQDeliveryResponseStatus{
							ID:   finishId,
							Name: RabbitMQMessageDeliveryStatus{}.Display(finishId),
						},
						Result: response,
					}

					push := RabbitMQ{
						Connection: messageDelivery.Connection,
						Queue:      queue,
						SenderId:   &messageDelivery.SenderId,
						SenderType: &messageDelivery.SenderType,
						Data:       deliveryRes,
					}
					push.Push()
				}
			}
		}
	}

	return pushNotification
}

func fanoutConsumer(ch *amqp091.Channel, connection xtrememodel.RabbitMQConnection, opt RabbitMQConsumeOpt) {
	err := ch.ExchangeDeclare(
		opt.Exchange,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare exchange %s: %s", opt.Exchange, err)
	}

	q, err := ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare a queue: %s", err)
	}

	err = ch.QueueBind(
		q.Name,
		"",
		opt.Exchange,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to bind queue %s to exchange %s: %s", q.Name, opt.Exchange, err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to register a consumer: %s", err)
	}

	go func() {
		for d := range msgs {
			process(connection, opt, d.Body)
		}
	}()
}

func directConsumer(ch *amqp091.Channel, connection xtrememodel.RabbitMQConnection, opt RabbitMQConsumeOpt) {
	q, err := ch.QueueDeclare(
		opt.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare a queue: %s", err)
	}

	err = ch.Qos(
		1,
		0,
		false,
	)
	if err != nil {
		log.Panicf("Failed to set QoS: %s", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to register a consumer: %s", err)
	}

	go func() {
		for d := range msgs {
			process(connection, opt, d.Body)
		}
	}()
}

func process(connection xtrememodel.RabbitMQConnection, opt RabbitMQConsumeOpt, body []byte) {
	var consumerKey string
	if opt.Queue != "" {
		consumerKey = opt.Queue
	} else if opt.Exchange != "" {
		consumerKey = opt.Exchange
	} else {
		consumerKey = "CONSUMER_DOES_NOT_EXISTS"
	}

	log.Printf("CONSUMING: %s %s", printMessage(consumerKey), time.DateTime)

	var mqBody rabbitMQBody
	err := json.Unmarshal(body, &mqBody)
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error unmarshalling: %s", err), true)
		return
	}

	var message xtrememodel.RabbitMQMessage
	err = RabbitMQSQL.First(&message, mqBody.MessageId).Error
	if err != nil {
		failed(connection, opt, mqBody, fmt.Sprintf("Get message data: %s", err), nil)
		return
	}

	result, err := opt.Consumer.Consume(message)
	if err != nil {
		failed(connection, opt, mqBody, fmt.Sprintf("Consume message is failed: %s", err), &message)
		return
	}

	finish(message)

	updateMessageDeliveryStatus(connection, &message, result, true)

	log.Printf("%-10s %s %s", "SUCCESS:", printMessage(consumerKey), time.DateTime)
}

func finish(message xtrememodel.RabbitMQMessage) {
	message.Finished = true

	err := RabbitMQSQL.Save(&message).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Update message status is failed: %s", err), false)
	}
}

func failed(connection xtrememodel.RabbitMQConnection, opt RabbitMQConsumeOpt, mqBody rabbitMQBody, errorMsg string, message *xtrememodel.RabbitMQMessage) {
	xtremepkg.LogError(errorMsg, true)

	exceptionRes := map[string]interface{}{"message": errorMsg, "trace": ""}

	payload, _ := json.Marshal(mqBody.Data)

	var messageFailed xtrememodel.RabbitMQMessageFailed
	messageFailed.ConnectionId = connection.ID
	messageFailed.MessageId = mqBody.MessageId
	messageFailed.Service = xtremepkg.GetServiceName()
	messageFailed.Exchange = opt.Exchange
	messageFailed.Queue = opt.Queue
	messageFailed.Payload = payload
	messageFailed.Exception = exceptionRes

	err := RabbitMQSQL.Create(&messageFailed).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Save message failed failed: %s", err), false)
	}

	updateMessageDeliveryStatus(connection, message, exceptionRes, false)
}

func updateMessageDeliveryStatus(connection xtrememodel.RabbitMQConnection, message *xtrememodel.RabbitMQMessage, result interface{}, isSuccess bool) {
	if message != nil && message.ID > 0 {
		var delivery xtrememodel.RabbitMQMessageDelivery
		RabbitMQSQL.Where("messageId = ?", message.ID).
			Where("consumerService = ?", xtremepkg.GetServiceName()).
			First(&delivery)
		if delivery.ID > 0 {
			deliveryResponses := make([]map[string]interface{}, 0)
			if delivery.Responses != nil {
				deliveryResponses = *delivery.Responses
			}

			resultMap := result.(map[string]interface{})
			deliveryResponses = append(deliveryResponses, resultMap)

			delivery.StatusId = RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID
			if isSuccess {
				delivery.StatusId = RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH_ID
			}

			delivery.Responses = (*xtrememodel.ArrayMapInterfaceColumn)(&deliveryResponses)

			RabbitMQSQL.Save(&delivery)

			if !delivery.NeedNotification {
				return
			}

			if message.Resend > 0 && delivery.StatusId == RABBITMQ_MESSAGE_DELIVERY_STATUS_ERROR_ID {
				return
			}

			queue := ""
			if message.Exchange != "" {
				queue = setQueueKey(message.Exchange)
			} else if message.Queue != "" {
				queue = setQueueKey(message.Queue)
			}

			if queue != "" {
				deliveryRes := RabbitMQDeliveryResponse{
					Status: rabbitMQDeliveryResponseStatus{
						ID:   delivery.StatusId,
						Name: RabbitMQMessageDeliveryStatus{}.Display(delivery.StatusId),
					},
				}

				if delivery.StatusId == RABBITMQ_MESSAGE_DELIVERY_STATUS_FINISH_ID {
					deliveryRes.Result = result
				} else {
					deliveryRes.Error = rabbitMQDeliveryResponseError{
						Message: resultMap["message"].(string),
						Trace:   resultMap["trace"].(string),
					}
				}

				push := RabbitMQ{
					Connection: connection.Connection,
					Queue:      queue,
					SenderId:   message.SenderId,
					SenderType: message.SenderType,
					Data:       deliveryRes,
				}
				push.Push()
			}
		}
	}
}

func setQueueKey(key string) string {
	keys := strings.Split(key, ".")

	lastKey := len(keys) - 1
	keys[lastKey] = "processed"

	keys = append(keys, "queue")

	return strings.Join(keys, ".")
}

func printMessage(message string) string {
	paddedStr := fmt.Sprintf("%-60s", message)
	return strings.ReplaceAll(paddedStr, " ", ".")
}
