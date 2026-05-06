package handlers

import (
	"fmt"
	"net"
)

func Redirect(conn net.Conn, target string) {
	resp := fmt.Sprintf("HTTP/1.1 302 Found\r\nLocation: %s\r\nContent-Length: 0\r\n\r\n", target)
	conn.Write([]byte(resp))
}
