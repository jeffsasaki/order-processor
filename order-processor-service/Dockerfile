FROM golang:1.16

WORKDIR /app
COPY . .

RUN go mod init order-processor-service
RUN go get -d -v ./...
RUN go install -v ./...

CMD ["order-processor-service"]
