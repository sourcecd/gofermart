package main

import (
	"context"
	"log"

	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/server"
)

func main() {
	ctx := context.Background()

	var config config.Config

	servFlags(&config)
	servEnv(&config)

	log.Fatal(config.Accu)

	server.Run(ctx, config)
}
