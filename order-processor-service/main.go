package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type Customer struct {
	ID        int    `json:"id,omitempty"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type Order struct {
	ID       int      `json:"id"`
	Customer Customer `json:"customer"`
	Product  string   `json:"product"`
	Status   string   `json:"status"`
}

type PaymentUpdate struct {
	OrderID       int    `json:"order_id"`
	PaymentStatus string `json:"payment_status"`
}

var db *sql.DB
var rabbitConn *amqp.Connection

func main() {
	var err error
	db, err = sql.Open("postgres", "host=postgres port=5432 user=user password=password dbname=db sslmode=disable")
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	rabbitConn, err = amqp.Dial("amqp://guest:guest@rabbitmq/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitConn.Close()

	// Setup RabbitMQ consumer
	setupPaymentUpdateListener()

	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/order", handleOrderSubmission)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupPaymentUpdateListener() {
	ch, err := rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"payment_updates", // name of the queue
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	go func() {
		for d := range msgs {
			var update PaymentUpdate
			if err := json.Unmarshal(d.Body, &update); err != nil {
				log.Printf("Error decoding message: %v", err)
				continue
			}
			log.Printf("Received payment update: %+v", update)

			// Update the order status based on the payment status
			_, err := db.Exec("UPDATE orders SET status = $1 WHERE id = $2", update.PaymentStatus, update.OrderID)
			if err != nil {
				log.Printf("Failed to update order status: %v", err)
				continue
			}

			d.Ack(false) // Acknowledge the message after processing
		}
	}()
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT o.id, o.product, o.status, c.id, c.first_name, c.last_name, c.email FROM orders o JOIN customers c ON o.customer_id = c.id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Product, &order.Status, &order.Customer.ID, &order.Customer.FirstName, &order.Customer.LastName, &order.Customer.Email); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		orders = append(orders, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func handleOrderSubmission(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// First, insert or update the customer in the database
	var customerID int
	err := db.QueryRow("INSERT INTO customers (first_name, last_name, email) VALUES ($1, $2, $3) ON CONFLICT (email) DO UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name RETURNING id", order.Customer.FirstName, order.Customer.LastName, order.Customer.Email).Scan(&customerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Then, insert the order with the customer_id
	_, err = db.Exec("INSERT INTO orders (customer_id, product, status) VALUES ($1, $2, 'Pending')", customerID, order.Product)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ch, err := rabbitConn.Channel()
	if err != nil {
		http.Error(w, "Failed to open a channel", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	order.Customer.ID = customerID
	body, _ := json.Marshal(order)
	err = ch.Publish(
		"",            // exchange
		"order_queue", // routing key
		false,         // mandatory
		false,         // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
	if err != nil {
		http.Error(w, "Failed to publish a message", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
