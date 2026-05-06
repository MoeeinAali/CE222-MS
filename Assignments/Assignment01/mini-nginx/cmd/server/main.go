package main

import (
	"Mini-NginX/internal/config"
	"Mini-NginX/internal/logger"
	"Mini-NginX/internal/server"
	"log"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatal(err)
	}

	logger.Init("server.log")

	s := server.New(cfg)
	logger.Info("Server Starting", "port", cfg.ListenOn)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
