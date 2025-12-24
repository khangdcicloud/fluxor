module github.com/quadgatefoundation/fluxor/examples/fluxor-project/api-gateway

go 1.24.0

require (
	github.com/fluxorio/fluxor v0.0.0
	github.com/quadgatefoundation/fluxor/examples/fluxor-project/common v0.0.0
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/nats-io/nats.go v1.48.0 // indirect
	github.com/nats-io/nkeys v0.4.12 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.68.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/quadgatefoundation/fluxor/examples/fluxor-project/common => ../common

// Use local framework checkout (this repo root)
replace github.com/fluxorio/fluxor => ../../..
