package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
	// It("should handle order submission correctly", func() {
	// 	// Prepare the data to be posted
	// 	order := models.Order{
	// 		Customer: models.Customer{
	// 			FirstName: "John",
	// 			LastName:  "Doe",
	// 			Email:     "johndoe@example.com",
	// 		},
	// 		Products: []models.Product{
	// 			{ProductID: 1, Name: "Widget", Price: 19.99},
	// 		},
	// 	}
	// 	body, _ := json.Marshal(order)

	// 	req, err := http.NewRequest("POST", "/order", bytes.NewBuffer(body))
	// 	Expect(err).NotTo(HaveOccurred())

	// 	rr := httptest.NewRecorder()
	// 	handler := http.HandlerFunc(HandleOrderSubmission)

	// 	// Mock database interactions
	// 	tx, _ := db.Begin()
	// 	dbMock.ExpectBegin()
	// 	dbMock.ExpectQuery(`INSERT INTO customers`).WithArgs("John", "Doe", "johndoe@example.com").WillReturnRows(sqlmock.NewRows([]string{"customer_id"}).AddRow(1))
	// 	dbMock.ExpectQuery(`INSERT INTO orders`).WithArgs(1, 19.99).WillReturnRows(sqlmock.NewRows([]string{"order_id"}).AddRow(1))
	// 	dbMock.ExpectExec(`INSERT INTO order_products`).WithArgs(1, 1).WillReturnResult(sqlmock.NewResult(1, 1))
	// 	dbMock.ExpectCommit()

	// 	// Assume a mock function for AMQP publishing that always succeeds
	// 	// (You would need to abstract this in your real code to properly mock it)
	// 	mockPublish := func(ch *amqp.Channel, order models.Order) error {
	// 		return nil // simulate successful publish
	// 	}

	// 	// Inject the mock publish function into your actual test environment
	// 	// (This requires modifying your HandleOrderSubmission to use a publish function passed in or set globally for the sake of testability)

	// 	handler.ServeHTTP(rr, req)

	// 	// Check the response code and body
	// 	Expect(rr.Code).To(Equal(http.StatusCreated))
	// })

})

func TestOrderProcessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OrderProcessor Suite")
}
