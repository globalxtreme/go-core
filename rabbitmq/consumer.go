package xtremerabbitmq

import (
	"github.com/rabbitmq/amqp091-go"
	"time"
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
	Name       string
	Type       string
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
