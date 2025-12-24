module github.com/quadgatefoundation/fluxor/fluxor-project/all-in-one

go 1.24.0

require (
	github.com/fluxorio/fluxor v0.0.0
	github.com/quadgatefoundation/fluxor/fluxor-project/api-gateway v0.0.0
	github.com/quadgatefoundation/fluxor/fluxor-project/payment-service v0.0.0
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.18.1 // indirect
	github.com/quadgatefoundation/fluxor/fluxor-project/common v0.0.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.68.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/quadgatefoundation/fluxor/fluxor-project/common => ../common

replace github.com/quadgatefoundation/fluxor/fluxor-project/api-gateway => ../api-gateway

replace github.com/quadgatefoundation/fluxor/fluxor-project/payment-service => ../payment-service

// Use local framework checkout (this repo root)
replace github.com/fluxorio/fluxor => ../..
