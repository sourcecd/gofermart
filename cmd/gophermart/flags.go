package main

import (
	"flag"
	"os"
	"strings"

	"github.com/sourcecd/gofermart/internal/config"
)

func servEnv(config *config.Config) {
	a := os.Getenv("RUN_ADDRESS")
	d := os.Getenv("DATABASE_URI")
	r := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")

	if a != "" {
		if len(strings.Split(a, ":")) == 2 {
			config.ServerAddr = a
		}
	}
	if d != "" {
		config.DatabaseDsn = d
	}
	if r != "" {
		if len(strings.Split(r, ":")) >= 2 {
			config.Accu = r
		}
	}
}

func servFlags(config *config.Config) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&config.DatabaseDsn, "d", "host=localhost database=gofermart sslmode=disable", "pg db connect address")
	flag.StringVar(&config.Accu, "r", "", "accu server")
	flag.Parse()
}
