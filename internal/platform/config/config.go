package config

import "github.com/caarlos0/env/v10"

type Config struct {
	HTTPPort string `env:"HTTP_PORT,notEmpty" envDefault:"8080"`
	DB       struct {
		DSN string `env:"DB_DSN"`
		URL string `env:"DATABASE_URL"`
	}
	LogLevel string `env:"LOG_LEVEL" envDefault:"info"`
}

func Load() Config {
	var c Config
	if err := env.Parse(&c); err != nil {
		panic(err)
	}
	if c.DB.DSN == "" {
		c.DB.DSN = c.DB.URL
	}
	return c
}
