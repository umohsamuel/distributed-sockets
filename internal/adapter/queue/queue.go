package queue

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/umohsamuel/distributed-sockets/internal/domain/queue"
)

type Queue struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

func NewQueueClient(conn *amqp.Connection, channel *amqp.Channel) queue.Interface {
	return &Queue{
		Conn:    conn,
		Channel: channel,
	}
}

func (q *Queue) Emit(exchangeConfig queue.ExchangeDeclare, publishConfig queue.Publish) error {
	err := q.Channel.ExchangeDeclare(
		exchangeConfig.Name,
		exchangeConfig.Kind,
		exchangeConfig.Durable,
		exchangeConfig.AutoDelete,
		exchangeConfig.Internal,
		exchangeConfig.NoWait,
		exchangeConfig.Args,
	)
	if err != nil {
		return err
	}

	return q.Channel.PublishWithContext(
		publishConfig.Ctx,
		publishConfig.Exchange,
		publishConfig.Key,
		publishConfig.Mandatory,
		publishConfig.Immediate,
		publishConfig.Msg,
	)
}

func (q *Queue) Recieve(exchangeConfig queue.ExchangeDeclare, queueDeclareConfig queue.QueueDeclare, queueBindConfig queue.QueueBind, consumerConfig queue.Consume, handler queue.MessageHandler) error {
	err := q.Channel.ExchangeDeclare(
		exchangeConfig.Name,
		exchangeConfig.Kind,
		exchangeConfig.Durable,
		exchangeConfig.AutoDelete,
		exchangeConfig.Internal,
		exchangeConfig.NoWait,
		exchangeConfig.Args,
	)
	if err != nil {
		return err
	}

	declaredQueue, err := q.Channel.QueueDeclare(
		queueDeclareConfig.Name,
		queueDeclareConfig.Durable,
		queueDeclareConfig.AutoDelete,
		queueDeclareConfig.Exclusive,
		queueDeclareConfig.NoWait,
		queueDeclareConfig.Args,
	)
	if err != nil {
		return err
	}

	if len(queueBindConfig.Key) < 1 {
		return fmt.Errorf("Queue bind key []string cannot be lesser than 1")
	}

	for _, key := range queueBindConfig.Key {
		err = q.Channel.QueueBind(
			declaredQueue.Name,
			key,
			queueBindConfig.Exchange,
			queueBindConfig.NoWait,
			queueBindConfig.Args,
		)

		if err != nil {
			return err
		}
	}

	msgs, err := q.Channel.Consume(
		declaredQueue.Name,
		consumerConfig.Consumer,
		consumerConfig.AutoAck,
		consumerConfig.Exclusive,
		consumerConfig.NoLocal,
		consumerConfig.NoWait,
		consumerConfig.Args,
	)
	if err != nil {
		return err
	}

	go func() {
		for msg := range msgs {
			if err := handler(msg.Body, msg); err != nil {
				log.Printf("Error handling message: %v", err)
			}
		}
	}()

	return nil
}
