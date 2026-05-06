package handlers

import (
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
)

func StaticFile(conn net.Conn, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		body := "file not found"
		resp := fmt.Sprintf("HTTP/1.1 404 Not Found\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
		conn.Write([]byte(resp))
		return
	}

	ext := filepath.Ext(path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	header := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: %s\r\n\r\n", len(data), mimeType)
	conn.Write([]byte(header))
	conn.Write(data)
}
