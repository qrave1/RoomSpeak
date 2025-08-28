package config

// Config holds application configuration.
type Config struct {
	Debug bool   `env:"DEBUG" envDefault:"true"`
	Port  string `env:"PORT" envDefault:"3000"`
}
