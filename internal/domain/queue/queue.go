package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ExchangeDeclare struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       amqp.Table
}

type QueueDeclare struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

type QueueBind struct {
	Name     string
	Key      []string
	Exchange string
	NoWait   bool
	Args     amqp.Table
}

type Consume struct {
	Queue     string
	Consumer  string
	AutoAck   bool
	Exclusive bool
	NoLocal   bool
	NoWait    bool
	Args      amqp.Table
}

type Publish struct {
	Ctx       context.Context
	Exchange  string
	Key       string
	Mandatory bool
	Immediate bool
	Msg       amqp.Publishing
}
