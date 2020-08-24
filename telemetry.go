package hats

import (
	"context"
	"net/http"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type telemetryServerInput struct {
	fx.In
	Viper     *viper.Viper
	Endpoints []httpEndpoint `group:"telemetryEndpoints"`
	Log       *zap.SugaredLogger
}

type NamedCheck struct {
	Name string
	Check healthcheck.Check
}

type healthCheckInputs struct {
	fx.In
	LivenessChecks []NamedCheck `group:"livenessChecks"`
	ReadinessChecks []NamedCheck `group:"readinessChecks"`
}

type healthCheckOutputs struct {
	fx.Out
	Ready httpEndpoint `group:"telemetryEndpoints"`
	Live  httpEndpoint `group:"telemetryEndpoints"`
}

func newHealth(inputs healthCheckInputs) healthCheckOutputs {
	hnd := healthcheck.NewHandler()
	for _, x := range inputs.LivenessChecks {
		hnd.AddLivenessCheck(x.Name, x.Check)
	}
	for _, x := range inputs.ReadinessChecks {
		hnd.AddReadinessCheck(x.Name, x.Check)
	}

	return healthCheckOutputs{
		Ready: httpEndpoint{"/readyz", http.HandlerFunc(hnd.LiveEndpoint) },
		Live:  httpEndpoint{"/healthz", http.HandlerFunc(hnd.ReadyEndpoint) },
	}
}

func newPromHttp(reg *prometheus.Registry) httpEndpoint {
	handler := promhttp.InstrumentMetricHandler(
		reg,
		promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}),
	)
	return httpEndpoint{"/metrics", handler}
}

func newTelemetryServer(in telemetryServerInput, lifecycle fx.Lifecycle) http.Server  {
	in.Viper.SetDefault("HatsTelemetryAddr", "127.0.0.1:8181")
	mux := http.NewServeMux()
	in.Log.Debugw("Going through endpoints", "endpoints", in.Endpoints)
	for _, v := range in.Endpoints {
		in.Log.Debugw("Added", "path", v.path)
		mux.Handle(v.path, v.handler)
	}
	server := http.Server{Addr: in.Viper.GetString("HatsTelemetryAddr"), Handler: mux}
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				err := server.ListenAndServe()
				if err != nil && err != http.ErrServerClosed {
					in.Log.Errorw("Error reported from server", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return server.Close()
		},
	})
	return server
}

type httpEndpoint struct {
	path string
	handler http.Handler
}

var telemetryModule = fx.Provide(
	newHealth,
	fx.Annotated{Name: "telemetryServer", Target: newTelemetryServer},
	prometheus.NewRegistry,
	fx.Annotated{Group: "telemetryEndpoints", Target: newPromHttp},
)


func RegisterPrometheusCollectors(collectors... prometheus.Collector) fx.Option {
	return fx.Invoke(func(registry *prometheus.Registry) error {
		for _, x := range collectors {
			if err := registry.Register(x); err != nil {
				return err
			}
		}
		return nil
	})
}