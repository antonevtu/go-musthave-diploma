package app

import (
	"fmt"
	"github.com/antonevtu/go-musthave-diploma/internal/cfg"
)

func Run() {
	cfg.New()
	fmt.Println("Hello world")
}
