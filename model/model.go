package model

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
