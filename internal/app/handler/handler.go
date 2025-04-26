package handler

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

type Handler struct{}

func handle(handler *Handler) (int, error) {
	l, err := net.Listen("tcp", "0.0.0.0:4221")
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	c, err := l.Accept()
	if err != nil {
		fmt.Println("Error accepting connection: ", err.Error())
		os.Exit(1)
	}

	remote := c.RemoteAddr()

	bytes := make([]byte, 1024)
	status, err := c.Read(bytes)
	if err != nil {
		log.Println(remote)
		return status, err
	}

	request := strings.Split(string(bytes), "\r\n")
	requestLine := strings.Split(request[0], " ")

	path := requestLine[1]

	if path == "/" {
		status, err = handler.Root(c)
		if err != nil {
			return status, err
		}
	} else if strings.HasPrefix(path, "/echo") {
		status, err = handler.Echo(c, requestLine)
		if err != nil {
			return status, err
		}
	} else {
		status, err := c.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		if err != nil {
			return status, err
		}
	}

	return status, err
}

func NewHandler() (int, error) {
	handler := Handler{}

	return handle(&handler)
}

func (h *Handler) Root(c net.Conn) (int, error) {
	status, err := c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		return status, err
	}

	return status, err
}

func (h *Handler) Echo(c net.Conn, requestLine []string) (int, error) {
	const FORMAT = "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: 3\r\n\r\n"

	body := strings.ReplaceAll(requestLine[1], "/echo/", "")

	echo := fmt.Sprintf(FORMAT+"%s", body)

	status, err := c.Write([]byte(echo))
	if err != nil {
		return status, err
	}

	return status, err
}
