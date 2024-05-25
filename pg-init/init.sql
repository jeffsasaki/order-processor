CREATE TABLE IF NOT EXISTS customers (
    customer_id SERIAL PRIMARY KEY,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) UNIQUE
);

CREATE TABLE IF NOT EXISTS products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE,
    price DECIMAL
);

INSERT INTO products (name, price) VALUES ('Cow', 4.99)
ON CONFLICT (name) DO NOTHING;

INSERT INTO products (name, price) VALUES ('Expensive Cow', 1001)
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS orders (
    order_id SERIAL PRIMARY KEY,
    customer_id INTEGER,
    amount DECIMAL,
    payment_status VARCHAR(255) DEFAULT 'Pending',
    FOREIGN KEY (customer_id) REFERENCES customers(customer_id),
);

CREATE TABLE IF NOT EXISTS order_products (
    order_products_id SERIAL PRIMARY KEY,
    order_id   INTEGER,
    product_id INTEGER,
    FOREIGN KEY (order_id) REFERENCES orders(order_id),
    FOREIGN KEY (product_id) REFERENCES products(product_id)
);