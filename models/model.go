package model

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
