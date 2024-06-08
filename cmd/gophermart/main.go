package main

import (
	"context"

	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/server"
)

func main() {
	ctx := context.Background()

	var config config.Config

	servFlags(&config)
	servEnv(&config)

	server.Run(ctx, config)
}
