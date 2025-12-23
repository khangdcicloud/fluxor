FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy root module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the example
RUN go build -o todo-api ./examples/todo-api/cmd/main.go

FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/todo-api .

EXPOSE 8080

CMD ["./todo-api"]
