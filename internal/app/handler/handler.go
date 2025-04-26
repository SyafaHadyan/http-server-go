package handler

import (
	"fmt"
	"net"
	"os"
)

type Handler struct{}

func NewHandler() error {
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

	err = handler.Root(c)
	if err != nil {
		return err
	}

	return nil
}

func (h *Handler) Root(c net.Conn) error {
	c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))

	return nil
}
