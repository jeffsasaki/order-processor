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

	// RabbitMQ consumer
	setupPaymentUpdateConsumer()

	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/order", handleOrderSubmission)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupPaymentUpdateConsumer() {
	ch, err := rabbitConn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	q, err := ch.QueueDeclare(
		"payment_update_queue",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	go func() {
		for d := range msgs {
			log.Printf("Message received: %s", d.Body)
			var update PaymentStatusUpdate
			if err := json.Unmarshal(d.Body, &update); err != nil {
				log.Printf("Error decoding message: %v", err)
				d.Nack(false, true)
				continue
			}

			log.Printf("Updating database for order ID %d with status %s", update.OrderID, update.PaymentStatus)
			_, err := db.Exec("UPDATE orders SET payment_status = $1 WHERE order_id = $2", update.PaymentStatus, update.OrderID)
			if err != nil {
				log.Printf("Database update failed: %v", err)
				d.Nack(false, true)
				continue
			}

			d.Ack(false)
			log.Printf("Order %d updated to status %s", update.OrderID, update.PaymentStatus)
		}
	}()
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
        SELECT
            o.order_id, o.amount,
            c.customer_id, c.first_name, c.last_name, c.email,
            o.payment_status,
            p.product_id, p.name, p.price
        FROM orders o
        JOIN customers c ON o.customer_id = c.customer_id
        JOIN order_products op ON o.order_id = op.order_id
        JOIN products p ON op.product_id = p.product_id
    `)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	orders := map[int]*Order{}
	for rows.Next() {
		var order Order
		var product Product
		var status string
		var customer Customer
		if err := rows.Scan(&order.ID, &order.Amount, &customer.ID, &customer.FirstName, &customer.LastName, &customer.Email, &status, &product.ProductID, &product.Name, &product.Price); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, ok := orders[order.ID]; !ok {
			orders[order.ID] = &Order{
				ID:       order.ID,
				Amount:   order.Amount,
				Customer: customer,
				Status:   status,
				Products: []Product{product},
			}
		} else {
			orders[order.ID].Products = append(orders[order.ID].Products, product)
		}
	}

	results := make([]Order, 0, len(orders))
	for _, order := range orders {
		results = append(results, *order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleOrderSubmission(w http.ResponseWriter, r *http.Request) {
	var order Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Upsert customer info
	var customerID int
	err = tx.QueryRow(`
        INSERT INTO customers (first_name, last_name, email)
        VALUES ($1, $2, $3)
        ON CONFLICT (email) DO 
            UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name 
            RETURNING customer_id`,
		order.Customer.FirstName, order.Customer.LastName, order.Customer.Email).Scan(&customerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var totalAmount float64 = 0
	for _, product := range order.Products {
		var price float64
		err = tx.QueryRow("SELECT price FROM products WHERE product_id = $1", product.ProductID).Scan(&price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		totalAmount += price
	}

	var orderID int
	err = tx.QueryRow(`
        INSERT INTO orders (customer_id, amount)
        VALUES ($1, $2) RETURNING order_id`,
		customerID, totalAmount).Scan(&orderID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, product := range order.Products {
		_, err = tx.Exec("INSERT INTO order_products (order_id, product_id) VALUES ($1, $2)", orderID, product.ProductID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Publish order for payment service to consume
	ch, err := rabbitConn.Channel()
	if err != nil {
		http.Error(w, "Failed to open a channel", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	order.Customer.ID = customerID
	order.ID = orderID
	order.Amount = float64(totalAmount)
	body, _ := json.Marshal(order)
	err = ch.Publish(
		"", "order_queue", false, false,
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
