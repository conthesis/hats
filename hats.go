package hats

import (
	"net/http"

	"github.com/nats-io/nats.go"
	_ "go.uber.org/automaxprocs"
	"go.uber.org/fx"
)

var Module = fx.Options(
	telemetryModule,
	LogModule,
	fx.Provide(newViper),
	fx.Invoke(background),
)

type backgroundInput struct {
	fx.In
	Server http.Server `name:"telemetryServer"`
	nats.Conn
}
func background(_ backgroundInput) {
	return
}

func RunWithHats(opts ...fx.Option) {
	added := append(opts, Module, ToggleFxLoggingToZap())
	app := fx.New(added...)
	app.Run()
}
