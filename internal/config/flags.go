package config

import (
	"flag"
	"log"
	"net"
	"net/url"
	"os"
)

func SetEnvironmentVariables(config *Config) {
	a := os.Getenv("RUN_ADDRESS")
	d := os.Getenv("DATABASE_URI")
	r := os.Getenv("ACCRUAL_SYSTEM_ADDRESS")

	if a != "" {
		if _, _, err := net.SplitHostPort(a); err != nil {
			log.Fatal("wrong server listen address")
		}
		config.ServerAddr = a
	}
	if d != "" {
		config.DatabaseDsn = d
	}
	if r != "" {
		if parsedURL, err := url.ParseRequestURI(r); err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			log.Fatal("wrong accrual system address")
		}
		config.AccrualSystemAddress = r
	}
}

func SetCmdlineFlags(config *Config) {
	flag.StringVar(&config.ServerAddr, "a", "localhost:8080", "Server bind addres and port")
	flag.StringVar(&config.DatabaseDsn, "d", "host=localhost database=gofermart sslmode=disable", "pg db connect address")
	flag.StringVar(&config.AccrualSystemAddress, "r", "", "accrual server")
	flag.Parse()
}
