package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/pion/webrtc/v4"
)

type Config struct {
	Debug       bool   `env:"DEBUG" envDefault:"false"`
	Port        string `env:"PORT" envDefault:"3000"`
	Domain      string `env:"DOMAIN" envDefault:"https://xxsm.ru"`
	PostgresURL string `env:"POSTGRES_URL,required"`

	TurnUDPServer webrtc.ICEServer
	TurnTCPServer webrtc.ICEServer

	CoturnServer CoturnConfig
}

type CoturnConfig struct {
	Host     string `env:"COTURN_HOST,required"`
	Username string `env:"COTURN_USERNAME,required"`
	Password string `env:"COTURN_PASSWORD,required"`

	// Secret - нужен для генерации временных кредов для фронта
	Secret string `env:"COTURN_SECRET,required"`
}

func New() (*Config, error) {
	c, err := env.ParseAs[Config]()
	if err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	c.TurnUDPServer = webrtc.ICEServer{
		URLs:       []string{fmt.Sprintf("turn:%s?transport=udp", c.CoturnServer.Host)},
		Username:   c.CoturnServer.Username,
		Credential: c.CoturnServer.Password,
	}

	c.TurnTCPServer = webrtc.ICEServer{
		URLs:       []string{fmt.Sprintf("turn:%s?transport=tcp", c.CoturnServer.Host)},
		Username:   c.CoturnServer.Username,
		Credential: c.CoturnServer.Password,
	}

	return &c, nil
}
