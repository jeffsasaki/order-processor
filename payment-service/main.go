package main

import (
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type Customer struct {
	ID        int    `json:"customer_id,omitempty"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type Order struct {
	ID       int       `json:"order_id"`
	Customer Customer  `json:"customer"`
	Products []Product `json:"products"`
	Amount   float64   `json:"amount"`
	Status   string    `json:"status,omitempty"`
}

type PaymentStatusUpdate struct {
	OrderID       int    `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
}

type Product struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name,omitempty"`
	Price     float64 `json:"price"`
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

	// Consumer
	q, err := ch.QueueDeclare(
		"order_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Producer - payment status update
	paymentUpdateQueue, err := ch.QueueDeclare(
		"payment_update_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare the order processor queue: %v", err)
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
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			var order Order
			if err := json.Unmarshal(d.Body, &order); err != nil {
				log.Printf("Error parsing order info: %v", err)
				continue
			}

			log.Printf("Order after unmarshal: %+v", order)

			totalAmount := order.Amount
			log.Printf("Total amount calculated: %f", totalAmount)

			var paymentStatus string
			if totalAmount > 1000 {
				paymentStatus = "Failure: Amount exceeds 1000"
			} else {
				paymentStatus = "Success"
			}

			log.Printf("Determined payment status: %s", paymentStatus)

			statusUpdate := PaymentStatusUpdate{
				OrderID:       order.ID,
				PaymentStatus: paymentStatus,
			}

			statusUpdateBytes, err := json.Marshal(statusUpdate)
			if err != nil {
				log.Printf("Error marshaling status update: %v", err)
				continue
			}

			err = ch.Publish(
				"",
				paymentUpdateQueue.Name,
				false,
				false,
				amqp.Publishing{
					ContentType: "application/json",
					Body:        statusUpdateBytes,
				},
			)
			if err != nil {
				log.Printf("Failed to publish order status update: %v", err)
			} else {
				log.Printf("Successfully published order status update: %s", string(statusUpdateBytes))
			}
		}
	}()

	log.Printf("Payment Service Started")
	<-forever
}
