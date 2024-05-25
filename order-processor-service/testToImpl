package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	model "order-processor-service/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Mocks for RabbitMQ components
type MockChannel struct {
	mock.Mock
}

func (m *MockChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	argsCall := m.Called(name, durable, autoDelete, exclusive, noWait, args)
	return argsCall.Get(0).(amqp.Queue), argsCall.Error(1)
}

func (m *MockChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	argsCall := m.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)
	return argsCall.Get(0).(<-chan amqp.Delivery), argsCall.Error(1)
}

func (m *MockChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	return m.Called(exchange, key, mandatory, immediate, msg).Error(0)
}

func (m *MockChannel) Close() error {
	return nil
}

type MockConnection struct {
	mock.Mock
}

func (m *MockConnection) Channel() (*MockChannel, error) {
	args := m.Called()
	return args.Get(0).(*MockChannel), args.Error(1)
}

func (m *MockConnection) Close() error {
	return nil
}

func TestHandleOrders(t *testing.T) {
	// Setup mock database connection
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	rows := sqlmock.NewRows([]string{"order_id", "amount", "customer_id", "first_name", "last_name", "email", "payment_status", "product_id", "name", "price"}).
		AddRow(1, 100.0, 1, "John", "Doe", "john@example.com", "Pending", 1, "Widget", 20.0)

	mock.ExpectQuery("^SELECT (.+) FROM orders").WillReturnRows(rows)

	// Set the global db variable to the mock DB
	oldDB := db
	db = mockDB
	defer func() { db = oldDB }() // Ensure the original db is restored after the test

	// Setup HTTP request and recorder
	req, err := http.NewRequest("GET", "/orders", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HandleOrders)

	// Perform the request
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect
	assert.Equal(t, http.StatusOK, rr.Code, "handler returned wrong status code")

	// Check the response body is what we expect
	expected := []model.Order{
		{
			ID: 1,
			Customer: model.Customer{
				ID:        1,
				FirstName: "John",
				LastName:  "Doe",
				Email:     "john@example.com",
			},
			Products: []model.Product{
				{
					ProductID: 1,
					Name:      "Widget",
					Price:     20.0,
				},
			},
			Amount: 100.0,
			Status: "Pending",
		},
	}
	var got []model.Order
	err = json.Unmarshal(rr.Body.Bytes(), &got)
	assert.NoError(t, err)
	assert.Equal(t, expected, got, "handler returned unexpected body")
}
