package handlers

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func ServeDir(conn net.Conn, dir string, route string, requestPath string) {
	decodedPath, err := url.PathUnescape(requestPath)
	if err != nil {
		write404(conn)
		return
	}

	rel := strings.TrimPrefix(decodedPath, route)
	rel = strings.TrimPrefix(rel, "/")
	fsPath := filepath.Join(dir, rel)

	info, err := os.Stat(fsPath)
	if err != nil {
		write404(conn)
		return
	}

	if !info.IsDir() {
		StaticFile(conn, fsPath)
		return
	}

	entries, err := os.ReadDir(fsPath)

	if err != nil {
		body := "cannot read directory"
		resp := fmt.Sprintf("HTTP/1.1 500 Internal Server Erorr\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
		conn.Write([]byte(resp))
		return
	}

	breadcrumb := "/" + strings.Trim(decodedPath, "/")

	parent := decodedPath
	if !strings.HasSuffix(parent, "/") {
		parent += "/"
	}
	parentBase := strings.TrimSuffix(parent, "/")
	parentEnc := encodeURLPathSegments(parentBase)

	var entriesHTML strings.Builder
	if len(entries) == 0 {
		entriesHTML.WriteString(`<div class="empty-state"><p>📭 This directory is empty</p></div>`)
	} else {
		entriesHTML.WriteString(`<ul class="entries-list">`)
		for _, e := range entries {
			name := e.Name()
			link := parentEnc + "/" + url.PathEscape(name)
			var entryType, icon, itemClass string

			if e.IsDir() {
				link = link + "/"
				entryType = "Folder"
				icon = "📁"
				itemClass = "entry-item folder-item"
			} else {
				entryType = "File"
				icon = "📄"
				itemClass = "entry-item file-item"
			}

			entriesHTML.WriteString(fmt.Sprintf(`
					<li><a href="%s" class="%s">
                    <span class="entry-icon">%s</span>
                    <span class="entry-name">%s</span>
                    <span class="entry-type">%s</span></a></li>`, link, itemClass, icon, escapeHTML(name), entryType))
		}
		entriesHTML.WriteString(`</ul>`)
	}

	templatePath := "static/dirlist.html"
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		body := "error loading template"
		resp := fmt.Sprintf("HTTP/1.1 500 Internal Server Error\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
		conn.Write([]byte(resp))
		return
	}

	html := string(templateData)
	html = strings.ReplaceAll(html, "{{BREADCRUMB}}", escapeHTML(breadcrumb))
	html = strings.ReplaceAll(html, "{{ENTRIES}}", entriesHTML.String())

	resp := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: text/html; charset=utf-8\r\n\r\n%s", len(html), html)
	conn.Write([]byte(resp))
}

func encodeURLPathSegments(decoded string) string {
	decoded = path.Clean("/" + decoded)
	segs := strings.Split(strings.Trim(decoded, "/"), "/")
	if len(segs) == 1 && segs[0] == "" {
		return "/"
	}
	esc := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "" {
			continue
		}
		esc = append(esc, url.PathEscape(s))
	}
	return "/" + strings.Join(esc, "/")
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

func write404(conn net.Conn) {
	body := "404 not found"
	resp := fmt.Sprintf("HTTP/1.1 404 Not Found\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	conn.Write([]byte(resp))
}
