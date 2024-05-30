package main

import (
	"context"

	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/server"
)

func main() {
	ctx := context.Background()
	config := &config.Config{
		Dsn: "host=localhost database=gofermart sslmode=disable",
	}

	server.Run(ctx, config)
}
