package server

import (
	"Mini-NginX/internal/config"
	"Mini-NginX/internal/logger"
	"Mini-NginX/internal/router"
	"fmt"
	"net"
)

type Server struct {
	cfg *config.Config
}

func New(cfg *config.Config) *Server {
	return &Server{cfg: cfg}
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.ListenOn))
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		logger.Debug("connection", "remote", conn.RemoteAddr().String())
		go handle(conn, s.cfg)
	}
}

func handle(conn net.Conn, cfg *config.Config) {
	defer conn.Close()
	router.Handle(conn, cfg)
}
