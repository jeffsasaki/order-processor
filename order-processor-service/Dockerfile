FROM golang:1.22

WORKDIR /app
COPY . .

RUN go mod init order-processor-service
RUN go clean -modcache
RUN go get -d -v ./...
RUN go install -v ./...

CMD ["order-processor-service"]
