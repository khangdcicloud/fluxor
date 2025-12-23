FROM golang:1.24 AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build the demo server (cmd/main.go)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/fluxor-demo ./cmd

FROM gcr.io/distroless/static:nonroot

COPY --from=builder /out/fluxor-demo /app/fluxor-demo

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/fluxor-demo"]
