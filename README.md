# Simple Order Processor
---
A simple order processing application comprising an order processing service and a payment service. The order processor communicates with the payment service (and vice versa) via a message queue to asynchronously fire and consume events, based on “purchase” criteria.

## Usage
The application consists of four components:
* order-processor-service
* payment-service
* Postgres
* RabbitMQ

To run the application in a containerized environment, ensure Docker is installed. From the root directory, simply run:
```
docker-compose up --build
```

The application will be exposed on http://localhost:8080/. After running `docker-compose up --build`, this will insert two products into the products table:
* Product ID 1: "Cow" for 4.99
* Product ID 2: "Expensive Cow" for 1001.00

## Example curl calls:
### POST /order
```
curl --location 'http://localhost:8080/order' \
--header 'Content-Type: application/json' \
--data '{
    "customer": {
        "first_name": "alice",
        "last_name": "bob",
        "email": "alice@bob.com"
    },
    "products": [
        { "product_id": 1 },
        { "product_id": 2 }
    ]
}
'

curl --location 'http://localhost:8080/order' \
--header 'Content-Type: application/json' \
--data '{
    "customer": {
        "first_name": "alice",
        "last_name": "bob",
        "email": "alice@bob.com"
    },
    "products": [
        { "product_id": 1 }
    ]
}
'
```

### GET /orders
```
curl --location 'http://localhost:8080/orders'
```

## Functional Requirements

- Web microservice for order management (just HTTP API, no web forms)
- Postgres database
- Another microservice for payment processing
- Some sort of messaging broker for asynchronous communication between two microservices

## Technical Specifications

### Order Processor Service
Written in Go, and acts as both an API and a message-driven application. The order processor has two endpoints:
* GET /orders - Shows all orders. Returns 200 on Success
* POST /order - Create an order. Returns 201 on Success (No data returned).

### Payment Service
Written in Go, which consumes events of orders made and determine if payment status will be "Success" or "Failure", given the total purchase amount.

## Database Schema

![Database Schema](https://i.ibb.co/d7WmD9h/Screenshot-2024-05-24-at-7-41-07-PM.png)

## Improvements

| Areas of Improvement | Solution |
|----------------------|----------|
| Use of plaintext credentials in code. | Pull from secrets, and sanitize commit history. |
| Prone to SQL Injection; not tested thoroughly. Only tested one basic query. | Sanitize user inputs and test for injections. |
| No audit logs or payment logs for database stores. | Create trigger and audit table for any upsert / delete queries. |
| Improve readability of code. Some functions are very long. | Move some code out into util functions. |
| Provide better line coverage for unit tests. AMQP mocking proved to be a challenging feat for the given timeframe. | Add more unit tests and separate logic for better coverage. |
| No CICD / linting implemented | Add workflow and golint (or alternative linter) |
| Duplication of models subdirectory | Ideally would like to keep this in a separate "common library" repository |
| Better error handling. If a product ID cannot be found, users are given an SQL error | Ideally would like to keep this in a separate "common library" repository |