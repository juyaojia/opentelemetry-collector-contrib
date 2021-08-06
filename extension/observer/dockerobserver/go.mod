module github.com/signalfx/forks/opentelemetry-collector-contrib/extension/observer/dockerobserver

go 1.16

require (
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/collector v0.31.1-0.20210805221026-cebe5773b96f
	go.opentelemetry.io/collector/model v0.31.1-0.20210805221026-cebe5773b96f // indirect
	go.uber.org/zap v1.18.1
)

replace github.com/open-telemetry/opentelemetry-collector-contrib/extension/observer => ../