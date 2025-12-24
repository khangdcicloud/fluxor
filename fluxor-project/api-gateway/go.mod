module github.com/quadgatefoundation/fluxor/fluxor-project/api-gateway

go 1.24.0

require (
	github.com/quadgatefoundation/fluxor/fluxor-project/common v0.0.0
	github.com/fluxorio/fluxor v0.0.0
)

replace github.com/quadgatefoundation/fluxor/fluxor-project/common => ../common

// Use local framework checkout (this repo root)
replace github.com/fluxorio/fluxor => ../..

