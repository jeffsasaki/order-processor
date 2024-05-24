package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/jeffsasaki/order-processor/models"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
)

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
			var update models.PaymentStatusUpdate
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
	rows, err := db.Query(`
        SELECT
            o.order_id,
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

	orders := map[int]*models.Order{}
	for rows.Next() {
		var orderID int
		var product models.Product
		var status string
		var customer models.Customer
		if err := rows.Scan(&orderID, &customer.ID, &customer.FirstName, &customer.LastName, &customer.Email, &status, &product.ProductID, &product.Name, &product.Price); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("OrderID: %d, CustomerID: %d, FirstName: %s, LastName: %s, Email: %s, Status: %s, ProductID: %d, ProductName: %s, Price: %f",
			orderID, customer.ID, customer.FirstName, customer.LastName, customer.Email, status, product.ProductID, product.Name, product.Price)

		if _, ok := orders[orderID]; !ok {
			orders[orderID] = &models.Order{
				ID:       orderID,
				Customer: customer,
				Status:   status,
				Products: []models.Product{product}, // Initialize with the first product
			}
		} else {
			orders[orderID].Products = append(orders[orderID].Products, product)
		}
	}

	results := make([]models.Order, 0, len(orders))
	for _, order := range orders {
		results = append(results, *order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func handleOrderSubmission(w http.ResponseWriter, r *http.Request) {
	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var customerID int
	err := db.QueryRow(`
		INSERT INTO customers (first_name, last_name, email)
		VALUES ($1, $2, $3) ON CONFLICT (email)
		DO UPDATE SET first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name RETURNING customer_id`,
		order.Customer.FirstName, order.Customer.LastName, order.Customer.Email).Scan(&customerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var orderID int
	err = tx.QueryRow(`
		INSERT INTO orders (customer_id, payment_status)
		VALUES ($1, 'Pending') RETURNING order_id`,
		customerID).Scan(&orderID)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, product := range order.Products {
		_, err = tx.Exec(
			`INSERT INTO order_products (order_id, product_id)
			VALUES ($1, $2)`,
			orderID, product.ProductID)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	err = tx.Commit()
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
	order.ID = orderID
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
