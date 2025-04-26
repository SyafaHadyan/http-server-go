package handler

import (
	"fmt"
	"net"
	"os"
)

type Handler struct{}

func NewHandler() (int, error) {
	handler := Handler{}

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

	status, err := handler.Root(c)
	if err != nil {
		return status, err
	}

	return status, err
}

func (h *Handler) Root(c net.Conn) (int, error) {
	status, err := c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
	if err != nil {
		return status, err
	}

	return status, err
}
