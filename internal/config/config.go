package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/pion/webrtc/v4"
)

type Config struct {
	Debug  bool   `env:"DEBUG" envDefault:"false"`
	Port   string `env:"PORT" envDefault:"3000"`
	Domain string `env:"DOMAIN" envDefault:"https://xxsm.ru"`

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

// TODO: на будущее для своего turn сервера
//type TurnServerConfig struct {
//	PublicIP string `env:"PUBLIC_IP" envDefault:"0.0.0.0"`
//	Host     string `env:"TURN_HOST,required"`
//	Port     int    `env:"TURN_PORT" envDefault:"3478"`
//	Realm    string `env:"TURN_REALM" envDefault:"xxsm.ru"`
//	Username string `env:"TURN_USERNAME,required"`
//	Password string `env:"TURN_PASSWORD,required"`
//	CertFile string `env:"TURN_CERT_FILE" envDefault:"/etc/certs/tls.crt"`
//	KeyFile  string `env:"TURN_KEY_FILE" envDefault:"/etc/certs/tls.key"`
//}

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
