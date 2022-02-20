package cfg

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
	"strconv"
)

type Config struct {
	RunAddress           string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	DatabaseURI          string `env:"DATABASE_URI"`
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`

	SecretKey         string `env:"SECRET_KEY" envDefault:"secret_key"`
	TokenPeriodExpire int64  `env:"TOKEN_PERIOD_EXPIRE" envDefault:"240"`

	CtxTimeout int64 `env:"CTX_TIMEOUT" envDefault:"500"`
}

func New() (Config, error) {
	var cfg Config

	// Заполнение cfg значениями из переменных окружения, в том числе дефолтными значениями
	err := env.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	// Если заданы аргументы командной строки - перетираем значения переменных окружения
	flag.Func("a", "self run address", func(flagValue string) error {
		cfg.RunAddress = flagValue
		return nil
	})
	flag.Func("d", "postgres url", func(flagValue string) error {
		cfg.DatabaseURI = flagValue
		return nil
	})
	flag.Func("t", "context timeout in seconds", func(flagValue string) error {
		t, err := strconv.Atoi(flagValue)
		if err != nil {
			return fmt.Errorf("can't parse context timeout -t: %w", err)
		}
		cfg.CtxTimeout = int64(t)
		return nil
	})
	flag.Func("k", "secret key for authorization tokens", func(flagValue string) error {
		cfg.SecretKey = flagValue
		return nil
	})
	flag.Func("e", "authorization token expiration time in hours", func(flagValue string) error {
		cfg.DatabaseURI = flagValue
		return nil
	})

	flag.Parse()

	return cfg, err
}
