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

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	context.AfterFunc(ctx, func() {
		time.Sleep(interruptAfter * time.Second)
		log.Fatal("Interrupted by shutdown time exeeded!!!")
	})

	var config config.Config

	SetCmdlineFlags(&config)
	SetEnvironmentVariables(&config)

	server.Run(ctx, config)
}
