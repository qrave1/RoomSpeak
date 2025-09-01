package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Debug      bool   `env:"DEBUG" envDefault:"false"`
	Port       string `env:"PORT" envDefault:"3000"`
	Domain     string `env:"DOMAIN" envDefault:"https://xxsm.ru"`
	TurnServer TurnServerConfig
}

type TurnServerConfig struct {
	URL      string `env:"TURN_URL,required"`
	Username string `env:"TURN_USERNAME,required"`
	Password string `env:"TURN_PASSWORD,required"`
}

func New() (*Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	return &c, nil
}
