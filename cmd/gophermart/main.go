package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/sourcecd/gofermart/internal/config"
	"github.com/sourcecd/gofermart/internal/server"
)

const interruptAfter = 30

func loadConfiguration(cfg *config.Config) {
	config.SetCmdlineFlags(cfg)
	config.SetEnvironmentVariables(cfg)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	context.AfterFunc(ctx, func() {
		time.Sleep(interruptAfter * time.Second)
		log.Fatal("Interrupted by shutdown time exeeded!!!")
	})

	var cfg config.Config
	loadConfiguration(&cfg)

	server.Run(ctx, cfg)
}
