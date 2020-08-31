package hats

import (
	"context"
	"fmt"
	"net/url"

	"github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.uber.org/fx"
)

type newNatsConnInput struct {
	fx.In
	Viper *viper.Viper
	Lifecycle fx.Lifecycle
	Handlers []SubjectHandlerOption `group:"natsHandlers"`
}

type SubjectHandlerOption interface {
	Apply(*nats.Conn) error
}

type SubjectMapHandlerOption struct {
	subjects map[string]nats.MsgHandler
}

func (s SubjectMapHandlerOption) Apply(conn *nats.Conn) error {
	for k, v := range s.subjects {
		if _, err := conn.Subscribe(k, v); err != nil {
			return err
		}
	}
	return nil
}

func NewSubjectMap(subjects map[string]nats.MsgHandler) SubjectHandlerOption {
	return SubjectMapHandlerOption{subjects}
}

func newNatsConn(in newNatsConnInput) (*nats.Conn, error) {
	natsUrl := in.Viper.GetString("NatsUrl")
	nc, err := nats.Connect(natsUrl)
	if err != nil {
		if err, ok := err.(*url.Error); ok {
			return nil, fmt.Errorf("NATS_URL is of an incorrect format: %w", err)
		}
		return nil, err
	}
	in.Lifecycle.Append(
		fx.Hook{
			OnStart: func(ctx context.Context) error {
				for _, x := range in.Handlers {
					if err := x.Apply(nc); err != nil {
						return err
					}
				}
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return nc.Drain()
			},
		})
	return nc, nil
}

func NatsHandlers(handlers ...interface{}) fx.Option {
	targets := make([]interface{}, 0, len(handlers))
	for _, h := range handlers {
		targets = append(targets, fx.Annotated{Group: "natsHandler", Target: h})
	}
	return fx.Provide(targets...)
}

var natsModule = fx.Provide(newNatsConn)