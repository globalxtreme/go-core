package command

import (
	"encoding/json"
	"fmt"
	"github.com/globalxtreme/go-core/model"
	xtremepkg "github.com/globalxtreme/go-core/pkg"
	"github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
	"log"
	"time"
)

type RabbitMQConsumeCommand struct {
	Channel *amqp091.Channel
}

type rabbitmqbody struct {
	MessageId uint   `json:"messageId"`
	Message   any    `json:"message"`
	Exchange  string `json:"exchange"`
	Queue     string `json:"queue"`
	Key       string `json:"key"`
}

func (class *RabbitMQConsumeCommand) Command(cmd *cobra.Command) {
	cmd.AddCommand(&cobra.Command{
		Use:  "rabbitmq-consume",
		Long: "RabbitMQ Consumer Command",
		Run: func(cmd *cobra.Command, args []string) {
			xtremepkg.InitDevMode()

			class.Handle()
		},
	})
}

func (class *RabbitMQConsumeCommand) Handle() {
	config := xtremepkg.RabbitMQConf
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
		exchange.Name,
		exchange.Type,
		exchange.Durable,
		exchange.AutoDelete,
		exchange.Internal,
		exchange.NoWait,
		exchange.Args,
	)
	if err != nil {
		log.Panicf("Failed to declare an exchange: %s", err)
	}

	q, err := ch.QueueDeclare(
		config.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to declare a queue: %s", err)
	}

	err = ch.QueueBind(
		q.Name,
		q.Name,
		exchange.Name,
		false,
		nil,
	)
	if err != nil {
		log.Panicf("Failed to bind a queue: %s", err)
	}

	msgs, err := ch.Consume(
		config.Queue,
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

	var forever chan struct{}

	go func() {
		for d := range msgs {
			processConsume(d.Body)
		}
	}()

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

func processConsume(body []byte) {
	log.Printf("CONSUMING:....................................  %s", time.DateTime)

	var mqBody rabbitmqbody
	err := json.Unmarshal(body, &mqBody)
	if err != nil {
		xtremepkg.Error(fmt.Sprintf("Error unmarshalling: %s", err))
		return
	}

	log.Printf("KEY: %s => %s", mqBody.Key, time.DateTime)

	var queueMessage model.RabbitMQMessage

	err = xtremepkg.RabbitMQSQL.First(&queueMessage, mqBody.MessageId).Error
	if err != nil {
		consumeInvalid(mqBody, fmt.Sprintf("Get message data: %s", err))
		return
	}

	if len(mqBody.Key) == 0 {
		consumeInvalid(mqBody, fmt.Sprintf("Your key invalid: %s", err))
		return
	}

	consumer := xtremepkg.RabbitMQConsumer{}.Get(mqBody.Key)
	if consumer == nil {
		consumeInvalid(mqBody, fmt.Sprintf("Your key does not exist: %s", err))
		return
	}

	err = consumer.Consume(mqBody.Message)
	if err != nil {
		consumeInvalid(mqBody, fmt.Sprintf("Consume message invalid: %s", err))
		return
	}

	updateMessageStatus(queueMessage)

	log.Printf("SUCCESS:....................................  %s", time.DateTime)
}

func updateMessageStatus(message model.RabbitMQMessage) {
	statuses := message.Statuses
	statuses[xtremepkg.RabbitMQConf.Queue] = true

	finished := true
	for _, status := range statuses {
		if !status {
			finished = false
			break
		}
	}

	message.Statuses = statuses
	message.Finished = finished

	err := xtremepkg.RabbitMQSQL.Save(&message).Error
	if err != nil {
		xtremepkg.Error(fmt.Sprintf("Update message status invalid: %s", err))
	}
}

func consumeInvalid(mqBody rabbitmqbody, message string) {
	xtremepkg.Error(message)

	payload, _ := json.Marshal(mqBody.Message)

	var messageFailed model.RabbitMQMessageFailed
	messageFailed.MessageId = mqBody.MessageId
	messageFailed.Sender = mqBody.Queue
	messageFailed.Consumer = xtremepkg.RabbitMQConf.Queue
	messageFailed.Key = mqBody.Key
	messageFailed.Payload = payload
	messageFailed.Exception = map[string]interface{}{"message": message, "trace": ""}

	err := xtremepkg.RabbitMQSQL.Save(&messageFailed).Error
	if err != nil {
		xtremepkg.Error(fmt.Sprintf("Save message failed invalid: %s", err))
	}
}
