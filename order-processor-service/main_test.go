package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	amqp "github.com/rabbitmq/amqp091-go"

	. "order-processor/models"
)

var _ = Describe("Order Processor", func() {
	var (
		dbMock sqlmock.Sqlmock
		err    error
	)

	BeforeEach(func() {
		var localDB *sql.DB
		localDB, dbMock, err = sqlmock.New() // Create mock database
		Expect(err).NotTo(HaveOccurred())
		db = localDB // Set the package level db variable to the mock instance
	})

	AfterEach(func() {
		db = nil                           // Reset the db variable after each test
		err = dbMock.ExpectationsWereMet() // Ensure all expectations were met
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("HandleOrders", func() {
		It("should fetch all orders successfully", func() {
			// Set up HTTP request and recorder
			req, err := http.NewRequest("GET", "/orders", nil)
			Expect(err).NotTo(HaveOccurred())
			rr := httptest.NewRecorder()

			handler := http.HandlerFunc(HandleOrders)

			// Set expectations on database mock
			rows := sqlmock.NewRows([]string{"order_id", "amount", "customer_id", "first_name", "last_name", "email", "payment_status", "product_id", "name", "price"}).
				AddRow(1, 4.99, 1, "John", "Doe", "john@example.com", "Success", 1, "Cow", 4.99).
				AddRow(2, 1001.0, 1, "Alice", "Bob", "alice@bob.com", "Failure: Amount is over 1000", 2, "Expensive Cow", 1001.0)
			dbMock.ExpectQuery(`SELECT`).WillReturnRows(rows)

			// Call the handler
			handler.ServeHTTP(rr, req)

			// Check the response body and HTTP status code
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(rr.Body.String()).To(ContainSubstring("John")) // Check for part of the expected output
		})
	})
})

var _ = Describe("Order Submission", func() {
	var (
		mockDB     *sql.DB
		dbMock     sqlmock.Sqlmock
		amqpClient *MockAmqpClient
		err        error
	)

	BeforeEach(func() {
		mockDB, dbMock, err = sqlmock.New()
		Expect(err).NotTo(HaveOccurred())
		db = mockDB // set the global db to the mock instance
		amqpClient = &MockAmqpClient{
			PublishFunc: func(body []byte, queueName string) error {
				// Assert that the message body and queue name are correct
				Expect(queueName).To(Equal("order_queue"))
				return nil
			},
		}
	})

	AfterEach(func() {
		db = nil
		err = dbMock.ExpectationsWereMet()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should handle order submission correctly", func() {
		// Prepare the order data
		order := Order{
			Customer: Customer{
				FirstName: "John",
				LastName:  "Doe",
				Email:     "johndoe@example.com",
			},
			Products: []Product{
				{ProductID: 1, Name: "Widget", Price: 19.99},
			},
		}
		body, _ := json.Marshal(order)
		req, err := http.NewRequest("POST", "/order", bytes.NewBuffer(body))
		Expect(err).NotTo(HaveOccurred())

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			HandleOrderSubmission(w, r, amqpClient)
		})

		// Mock the expected database operations
		dbMock.ExpectBegin()
		dbMock.ExpectQuery(`INSERT INTO customers`).WithArgs("John", "Doe", "johndoe@example.com").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(1))
		dbMock.ExpectQuery(`SELECT price FROM products WHERE product_id =`).WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"price"}).AddRow(19.99))
		dbMock.ExpectQuery(`INSERT INTO orders`).WithArgs(1, 19.99).WillReturnRows(sqlmock.NewRows([]string{"order_id"}).AddRow(1))
		dbMock.ExpectExec(`INSERT INTO order_products`).WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(1, 1))
		dbMock.ExpectCommit()

		// Execute the handler
		handler.ServeHTTP(rr, req)

		// Verify the response
		Expect(rr.Code).To(Equal(http.StatusCreated)) // Ensure the HTTP status code is 201
	})

	// AfterEach to verify all expectations were met
	AfterEach(func() {
		db = nil
		err = dbMock.ExpectationsWereMet()
		Expect(err).NotTo(HaveOccurred())
	})
})

func TestOrderProcessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrderProcessor Suite")
}

type MockAmqpClient struct {
	PublishFunc func(body []byte, queueName string) error
}

func (m *MockAmqpClient) Publish(body []byte, queueName string) error {
	if m.PublishFunc != nil {
		return m.PublishFunc(body, queueName)
	}
	return nil // Default to no operation if no function is set
}

func (m *MockAmqpClient) SetupConsumer(queueName string, handler func(d amqp.Delivery)) error {
	// Mock SetupConsumer if necessary
	return nil
}
