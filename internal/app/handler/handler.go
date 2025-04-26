package handler

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

type Handler struct {
	listener net.Listener
	conn     net.Conn
	serveDir string
}

func NewHandler(serveDir string) {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		handler := Handler{
			listener: l,
			conn:     c,
			serveDir: serveDir,
		}

		go handle(&handler)
	}
}

func handle(handler *Handler) (int, error) {
	remote := handler.conn.RemoteAddr()

	bytes := make([]byte, 1024)
	status, err := handler.conn.Read(bytes)
	if err != nil {
		log.Println(remote)
		return status, err
	}

	request := strings.Split(string(bytes), "\r\n")

	log.Println(strings.Join(request, ", "))

	path := strings.Split(request[0], " ")[1]

	switch strings.Split(request[0], " ")[0] {
	case "GET":
		if path == "/" {
			status, err = handler.Root()
			if err != nil {
				return status, err
			}
		} else if strings.HasPrefix(path, "/echo") {
			status, err = handler.Echo(request)
			if err != nil {
				return status, err
			}
		} else if strings.HasPrefix(path, "/user-agent") {
			status, err = handler.UserAgent(request)
			if err != nil {
				return status, err
			}
		} else if strings.HasPrefix(path, "/files") {
			status, err = handler.Files(request)
			if err != nil {
				return status, err
			}
		} else {
			status, err := handler.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
			if err != nil {
				return status, err
			}
		}
	case "POST":
		if strings.HasPrefix(path, "/files") {
			status, err = handler.NewFile(request)
			if err != nil {
				return status, err
			}
		}
	}

	return status, err
}

func (h *Handler) Root() (int, error) {
	status, err := h.conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		return status, err
	}

	h.conn.Close()

	return status, err
}

func (h *Handler) Echo(request []string) (int, error) {
	body := strings.Split(request[0], " ")[1]
	body = strings.ReplaceAll(body, "/echo/", "")

	echo := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		utf8.RuneCountInString(body),
		body,
	)

	status, err := h.conn.Write([]byte(echo))
	if err != nil {
		return status, err
	}

	h.conn.Close()

	return status, err
}

func (h *Handler) UserAgent(request []string) (int, error) {
	body := strings.ReplaceAll(request[2], "User-Agent: ", "")

	userAgent := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		utf8.RuneCountInString(body),
		body,
	)

	status, err := h.conn.Write([]byte(userAgent))
	if err != nil {
		return status, err
	}

	h.conn.Close()

	return status, err
}

func (h *Handler) Files(request []string) (int, error) {
	body := strings.Split(request[0], " ")[1]
	body = strings.ReplaceAll(body, "/files/", "")

	fileContent, err := os.ReadFile(h.serveDir + body)
	if err != nil {
		log.Println(err)

		status, err := h.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		if err != nil {
			log.Println(err)
		}

		return status, err
	}

	files := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
		len(fileContent),
		string(fileContent[:]),
	)

	status, err := h.conn.Write([]byte(files))
	if err != nil {
		return status, err
	}

	h.conn.Close()

	return status, err
}

func (h *Handler) NewFile(request []string) (int, error) {
	path := strings.Split(request[0], " ")[1]
	fileName := strings.TrimPrefix(path, "/files/")

	var fileContent string
	for i, line := range request {
		if line == "" && i+1 < len(request) {
			fileContent = strings.TrimSpace(request[i+1])
			break
		}
	}

	file, err := os.Create(h.serveDir + fileName)
	if err != nil {
		log.Println(err)
	}

	fileContentByte := bytes.Trim([]byte(fileContent), "\x00")

	err = os.WriteFile(filepath.Join(h.serveDir, fileName), fileContentByte, 0666)
	if err != nil {
		log.Println(err)
	}

	status, err := h.conn.Write([]byte("HTTP/1.1 201 Created\r\n\r\n"))
	if err != nil {
		return status, err
	}

	file.Sync()
	file.Close()
	h.conn.Close()

	return status, err
}
