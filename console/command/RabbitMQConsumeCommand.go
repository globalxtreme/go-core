package command

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	model2 "github.com/globalxtreme/go-core/v2/model"
	xtremepkg "github.com/globalxtreme/go-core/v2/pkg"
	xtremerabbitmq "github.com/globalxtreme/go-core/v2/rabbitmq"
	"github.com/rabbitmq/amqp091-go"
	"github.com/spf13/cobra"
)

type RabbitMQConsumeCommand struct {
	Channel *amqp091.Channel
	Config  []RabbitMQConsumeConfig
}

type RabbitMQConsumeConfig struct {
	Type     string
	Name     string
	RouteKey string
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
func (class *RabbitMQConsumeCommand) SetConfig(config []RabbitMQConsumeConfig) *RabbitMQConsumeCommand {
	class.Config = config
	return class
}

func (class *RabbitMQConsumeCommand) Handle() {
	config := xtremerabbitmq.RabbitMQConf
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

	var forever chan struct{}

	for _, configMap := range class.Config {
		err = ch.ExchangeDeclare(
			configMap.Name,
			configMap.Type,
			exchange.Durable,
			exchange.AutoDelete,
			exchange.Internal,
			exchange.NoWait,
			exchange.Args,
		)
		if err != nil {
			log.Panicf("Failed to declare exchange %s: %s", configMap.Name, err)
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
			configMap.RouteKey,
			configMap.Name,
			false,
			nil,
		)
		if err != nil {
			log.Panicf("Failed to bind queue %s to exchange %s: %s", q.Name, configMap.Name, err)
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
				processConsume(d.Body)
			}
		}()
	}

	log.Printf(" [*] Waiting for logs. To exit press CTRL+C")
	<-forever
}

func processConsume(body []byte) {
	log.Printf("CONSUMING:....................................  %s", time.DateTime)

	var mqBody rabbitmqbody
	err := json.Unmarshal(body, &mqBody)
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Error unmarshalling: %s", err), true)
		return
	}

	log.Printf("KEY: %s => %s", mqBody.Key, time.DateTime)

	var queueMessage model2.RabbitMQMessage

	err = xtremerabbitmq.RabbitMQSQL.First(&queueMessage, mqBody.MessageId).Error
	if err != nil {
		consumeInvalid(mqBody, fmt.Sprintf("Get message data: %s", err))
		return
	}

	if len(mqBody.Key) == 0 {
		consumeInvalid(mqBody, fmt.Sprintf("Your key invalid: %s", err))
		return
	}

	consumer := xtremerabbitmq.Consumer{}.Get(mqBody.Key)
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

func updateMessageStatus(message model2.RabbitMQMessage) {
	statuses := message.Statuses
	statuses[xtremerabbitmq.RabbitMQConf.Queue] = true

	finished := true
	for _, status := range statuses {
		if !status {
			finished = false
			break
		}
	}

	message.Statuses = statuses
	message.Finished = finished

	err := xtremerabbitmq.RabbitMQSQL.Save(&message).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Update message status invalid: %s", err), false)
	}
}

func consumeInvalid(mqBody rabbitmqbody, message string) {
	xtremepkg.LogError(message, true)

	payload, _ := json.Marshal(mqBody.Message)

	var messageFailed model2.RabbitMQMessageFailed
	messageFailed.MessageId = mqBody.MessageId
	messageFailed.Sender = mqBody.Queue
	messageFailed.Consumer = xtremerabbitmq.RabbitMQConf.Queue
	messageFailed.Key = mqBody.Key
	messageFailed.Payload = payload
	messageFailed.Exception = map[string]interface{}{"message": message, "trace": ""}

	err := xtremerabbitmq.RabbitMQSQL.Save(&messageFailed).Error
	if err != nil {
		xtremepkg.LogError(fmt.Sprintf("Save message failed invalid: %s", err), false)
	}
}
