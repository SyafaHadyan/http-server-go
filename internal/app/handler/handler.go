package handler

import (
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"unicode/utf8"
)

type Handler struct {
	listener net.Listener
	conn     net.Conn
}

func NewHandler() {
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

	if path == "/" {
		status, err = handler.Root(handler.conn)
		if err != nil {
			return status, err
		}
	} else if strings.HasPrefix(path, "/echo") {
		status, err = handler.Echo(handler.conn, request)
		if err != nil {
			return status, err
		}
	} else if strings.HasPrefix(path, "/user-agent") {
		status, err = handler.UserAgent(handler.conn, request)
		if err != nil {
			return status, err
		}
	} else {
		status, err := handler.conn.Write([]byte("HTTP/1.1 404 Not Found\r\n\r\n"))
		if err != nil {
			return status, err
		}
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

func (h *Handler) Echo(c net.Conn, request []string) (int, error) {
	body := strings.Split(request[0], " ")[1]
	body = strings.ReplaceAll(body, "/echo/", "")

	echo := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		utf8.RuneCountInString(body),
		body)

	status, err := c.Write([]byte(echo))
	if err != nil {
		return status, err
	}

	return status, err
}

func (h *Handler) UserAgent(c net.Conn, request []string) (int, error) {
	body := strings.ReplaceAll(request[2], "User-Agent: ", "")

	userAgent := fmt.Sprintf(
		"HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		utf8.RuneCountInString(body),
		body)

	status, err := c.Write([]byte(userAgent))
	if err != nil {
		return status, err
	}

	return status, err
}
