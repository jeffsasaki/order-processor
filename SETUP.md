# Setup
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

## Unit Tests
Unit test can be ran using 
```
go test
```
from the `order-processor-service` or `payment-service` directory.