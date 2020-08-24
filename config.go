package hats

import "github.com/spf13/viper"

func newViper() *viper.Viper {
	vp := viper.New()
	vp.BindEnv("NatsUrl", "NATS_URL")
	return vp
}
