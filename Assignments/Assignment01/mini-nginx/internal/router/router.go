package router

import (
	"Mini-NginX/internal/config"
	"Mini-NginX/internal/handlers"
	"Mini-NginX/internal/logger"
	"bufio"
	"fmt"
	"net"
	"strings"
)

func Handle(conn net.Conn, cfg *config.Config) {
	remote := conn.RemoteAddr().String()
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	parts := strings.SplitN(strings.TrimSpace(line), " ", 3)
	if len(parts) < 2 {
		return
	}
	method := parts[0]
	reqPath := parts[1]

	for route, rule := range cfg.Paths {
		if strings.HasPrefix(reqPath, route) {
			logger.Access(remote, method, reqPath, fmt.Sprintf("handler=%s route=%q", rule.Type, route))

			switch rule.Type {
			case "redirect":
				handlers.Redirect(conn, rule.Target)
			case "staticfile":
				handlers.StaticFile(conn, rule.Target)
			case "servedir":
				handlers.ServeDir(conn, rule.Target, route, reqPath)
			case "reverseproxy":
				handlers.ReverseProxy(conn, reader, rule.Target, line)
			default:
				write404(conn)
			}
			return
		}
	}

	logger.Access(remote, method, reqPath, "handler=none status=404")
	write404(conn)
}

func write404(conn net.Conn) {
	body := "404 not found"
	resp := fmt.Sprintf("HTTP/1.1 404 Not Found\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	conn.Write([]byte(resp))
}
