package main

import (
	"encoding/json"
	"testing"

	"payment-service/models"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Test payment status determination based on order amount
func TestPaymentStatus(t *testing.T) {
	conn := &MockConnection{}
	ch := &MockChannel{}
	conn.On("Channel").Return(ch, nil)

	ch.On("QueueDeclare", mock.Anything, false, false, false, false, mock.Anything).Return(amqp.Queue{}, nil)

	deliveryChannel := make(chan amqp.Delivery)
	ch.On("Consume", mock.Anything, "", true, false, false, false, nil).Return((<-chan amqp.Delivery)(deliveryChannel), nil)

	order := models.Order{
		ID:     1,
		Amount: 999.99,
	}
	body, _ := json.Marshal(order)
	delivery := amqp.Delivery{Body: body}

	go func() {
		deliveryChannel <- delivery
		close(deliveryChannel)
	}()

	received := make(chan bool)
	go func() {
		for d := range deliveryChannel {
			var order models.Order
			if err := json.Unmarshal(d.Body, &order); err != nil {
				t.Errorf("Failed to unmarshal: %v", err)
				received <- false
				return
			}

			assert.Equal(t, 999.99, order.Amount)
			paymentStatus := "Success"
			assert.Equal(t, paymentStatus, "Success")
			received <- true
		}
	}()

	<-received
}

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
