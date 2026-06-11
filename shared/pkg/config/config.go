// Package config provides a generic environment variable loader using Viper.
// Each service defines its own config struct and calls Load() to populate it.
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// Load reads .env file and environment variables, then unmarshals into dst.
// dst must be a pointer to a struct with mapstructure tags.
//
// Usage in any service:
//
//	type Config struct {
//	    BotToken string `mapstructure:"BOT_TOKEN"`
//	    DSN      string `mapstructure:"POSTGRES_DSN"`
//	}
//	var C Config
//	config.Load(&C)
func Load(dst any, envFile ...string) error {
	file := ".env"
	if len(envFile) > 0 && envFile[0] != "" {
		file = envFile[0]
	}

	viper.SetConfigFile(file)
	viper.AutomaticEnv()
	_ = viper.ReadInConfig() // ok if file is missing (pure env vars)

	if err := viper.Unmarshal(dst); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}
	return nil
}

// MustLoad is like Load but panics on error.
func MustLoad(dst any, envFile ...string) {
	if err := Load(dst, envFile...); err != nil {
		panic(err)
	}
}
