package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"github.com/streadway/amqp"
)

type Order struct {
	ID       int    `json:"id"`
	Customer string `json:"customer"`
	Product  string `json:"product"`
	Status   string `json:"status"`
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

	http.HandleFunc("/orders", handleOrders)
	http.HandleFunc("/order", handleOrderSubmission)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, customer, product, status FROM orders")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Customer, &order.Product, &order.Status); err != nil {
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

	_, err := db.Exec("INSERT INTO orders (customer, product, status) VALUES ($1, $2, 'Pending')", order.Customer, order.Product)
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
