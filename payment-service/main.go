package main

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type PaymentUpdate struct {
	OrderID       int    `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"order_queue", // name
		false,         // durable
		false,         // delete when unused
		false,         // exclusive
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			var paymentUpdate PaymentUpdate
			if err := json.Unmarshal(d.Body, &paymentUpdate); err != nil {
				log.Printf("Error parsing payment info: %v", err)
				continue
			}

			// Simulate payment processing
			paymentUpdate.PaymentStatus = "Processed"

			// Update order status in the database (assuming access to the DB or another service)
			log.Printf("Processed order %d, status %s", paymentUpdate.OrderID, paymentUpdate.PaymentStatus)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
