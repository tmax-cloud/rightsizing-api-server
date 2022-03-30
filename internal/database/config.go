package database

import "github.com/caarlos0/env/v6"

type envConfig struct {
	Host     string `env:"HOST"`
	Port     string `env:"PORT" envDefault:"5432"`
	User     string `env:"USER,unset" envDefault:"postgres"`
	Password string `env:"PASSWORD,unset"`
	DBName   string `env:"DATABASE" envDefault:"postgres"`
}

func NewConfig() (*envConfig, error) {
	dbConfig := &envConfig{}
	opts := env.Options{}
	if err := env.Parse(dbConfig, opts); err != nil {
		return nil, err
	}
	return dbConfig, nil
}
