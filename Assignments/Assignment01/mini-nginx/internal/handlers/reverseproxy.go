package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net"
)

func ReverseProxy(client net.Conn, reader *bufio.Reader, target string, firstLine string) {
	backend, err := net.Dial("tcp", target)
	if err != nil {
		resp := "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 11\r\n\r\nBad Gateway"
		client.Write([]byte(resp))
		return
	}
	defer backend.Close()
	fmt.Fprint(backend, firstLine)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Fprint(backend, line)
		if line == "\r\n" {
			break
		}
	}
	io.Copy(client, backend)
}
