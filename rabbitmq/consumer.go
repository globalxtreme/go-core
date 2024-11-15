package xtremerabbitmq

import (
	"time"

	"github.com/rabbitmq/amqp091-go"
)

var RabbitMQConsumer map[string]RabbitMQConsumerInterface

type rabbitmqconf struct {
	Queue      string
	Connection RabbitMQConnection
	Exchange   RabbitMQExchange
	Timeout    time.Duration
}

type RabbitMQConnection struct {
	Host     string
	Port     string
	Username string
	Password string
}

type RabbitMQExchange struct {
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp091.Table
}

type RabbitMQConsumerInterface interface {
	Consume(message any) error
}

type Consumer struct {
}

func (Consumer) Set(consumers map[string]RabbitMQConsumerInterface) {
	RabbitMQConsumer = consumers
}

func (Consumer) Get(key string) RabbitMQConsumerInterface {
	consumer := RabbitMQConsumer[key]
	return consumer
}
