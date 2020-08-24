package hats

import (
	"log"

	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func newZapLogger(v *viper.Viper) (*zap.Logger, error) {
	return zap.NewDevelopment()
}

func newSugaredLogger(logger *zap.Logger) *zap.SugaredLogger {
	return logger.Sugar()
}

type ToggledPrinter struct {
	logger *zap.SugaredLogger
}

func (tl *ToggledPrinter) ToggleTo(logger *zap.SugaredLogger) {
	tl.logger = logger.With("module", "Fx")
}

func (tl *ToggledPrinter) Printf(format string, v ...interface{}){
	if tl.logger != nil {
		tl.logger.Infof(format, v)
	} else {
		log.Printf(format, v)
	}
}

func ToggleFxLoggingToZap() fx.Option {
	tl := ToggledPrinter{nil}
	return fx.Options(
		fx.Logger(&tl),
		fx.Invoke(tl.ToggleTo),
	)
}

var LogModule fx.Option = fx.Options(
	fx.Provide(newZapLogger, newSugaredLogger),
	ToggleFxLoggingToZap(),
)
