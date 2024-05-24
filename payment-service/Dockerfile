FROM golang:1.22

WORKDIR /app
COPY . .

RUN go mod init payment-service
RUN go get -d -v ./...
RUN go install -v ./...

CMD ["payment-service"]
