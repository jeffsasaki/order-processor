# Simple Order Processor
-
A simple order processing application comprising an order processing service and a payment service. The order processor communicates with the payment service (and vice versa) via a message queue to asynchronously fire and consume events, based on “purchase” criteria.