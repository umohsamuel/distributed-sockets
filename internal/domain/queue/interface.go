package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(body []byte, mainMsg amqp.Delivery) error

type Interface interface {
	Emit(exchangeConfig ExchangeDeclare, publishConfig Publish) error

	Recieve(exchangeConfig ExchangeDeclare, queueDeclareConfig QueueDeclare, queueBindConfig QueueBind, consumerConfig Consume, handler MessageHandler) error
}
