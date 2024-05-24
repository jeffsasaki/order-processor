package amqp_client

import (
	"github.com/streadway/amqp"
)

// AmqpClient defines the interface for all AMQP operations
type AmqpClient interface {
	Publish(message []byte, queueName string) error
	SetupConsumer(queueName string, handler func(amqp.Delivery)) error
}

// RealAmqpClient implements AmqpClient with real AMQP operations
type RealAmqpClient struct {
	conn *amqp.Connection
}

// NewAmqpClient creates a new real AMQP client
func NewAmqpClient(conn *amqp.Connection) AmqpClient {
	return &RealAmqpClient{conn: conn}
}

// Publish publishes a message to a specified queue
func (c *RealAmqpClient) Publish(message []byte, queueName string) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	err = ch.Publish(
		"",        // exchange
		queueName, // routing key (queue name)
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        message,
		},
	)
	return err
}

// SetupConsumer sets up a consumer on a specified queue
func (c *RealAmqpClient) SetupConsumer(queueName string, handler func(amqp.Delivery)) error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		return err
	}

	go func() {
		for d := range msgs {
			handler(d)
		}
	}()

	return nil
}
