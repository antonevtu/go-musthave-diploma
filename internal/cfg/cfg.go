package cfg

type Config struct {
	Path string
}

func New() (Config, error) {
	return Config{}, nil
}
