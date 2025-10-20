package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/pion/webrtc/v4"
)

type Config struct {
	Debug     bool   `env:"DEBUG" envDefault:"false"`
	Port      string `env:"PORT" envDefault:"3000"`
	Domain    string `env:"DOMAIN" envDefault:"http://localhost:3000"`
	JWTSecret string `env:"JWT_SECRET,required"`

	TurnUDPServer webrtc.ICEServer
	TurnTCPServer webrtc.ICEServer

	CoturnServer CoturnConfig
	Postgres     PostgresConfig
}

type PostgresConfig struct {
	URL string `env:"POSTGRES_URL"`

	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER" envDefault:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	Name     string `env:"POSTGRES_NAME" envDefault:"roomspeak"`
	SSL      string `env:"POSTGRES_SSL" envDefault:"disable"`
}

func (p *PostgresConfig) DSN() string {
	if p.URL != "" {
		return p.URL
	}

	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		p.User,
		p.Password,
		p.Host,
		p.Port,
		p.Name,
		p.SSL,
	)
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
