package handler

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	supportedEncoding []string
	httpStatus        map[string]string
)

type Handler struct {
	listener net.Listener
	conn     net.Conn
	serveDir string
	request  []string
}

func init() {
	supportedEncoding = make([]string, 1)
	supportedEncoding[0] = "gzip"

	httpStatus = make(map[string]string)
	httpStatus["ok"] = "HTTP/1.1 200 OK\r\n"
	httpStatus["not found"] = "HTTP/1.1 404 Not Found\r\n"
	httpStatus["created"] = "HTTP/1.1 201 Created\r\n"
}

func (h *Handler) GetEncoding() string {
	for i := range h.request {
		current := h.request[i]

		if strings.HasPrefix(current, "Accept-Encoding: ") {
			current := strings.Split(strings.ReplaceAll(current, "Accept-Encoding: ", ""), ", ")
			for j := range current {
				if slices.Contains(supportedEncoding, current[j]) {
					return "Content-Encoding: " + current[j] + "\r\n"
				}
			}
		}
	}

	return "\r\n"
}

func (h *Handler) HandleCloseConnection() (string, bool) {
	for i := range h.request {
		if strings.Contains(h.request[i], "Connection: close") {
			return "Connection: close\r\n", true
		}
	}

	return "\r\n", false
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

func handle(h *Handler) {
	for {
		request := h.readRequest()
		if request == "" {
			break
		}

		h.handleBuilder(request)
	}

	err := h.conn.Close()
	if err != nil {
		log.Println(err)
	}
}

func (h *Handler) handleBuilder(request string) {
	requestLine := strings.Split(request, "\r\n")

	log.Println(strings.Join(requestLine, ", "))

	h.HandleRequest(requestLine)
}

func (h *Handler) readRequest() string {
	reader := bufio.NewReader(h.conn)
	var request strings.Builder

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		log.Println(err)
		return ""
	}
	request.WriteString(requestLine)

	var contentLength int
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}

		if strings.HasPrefix(line, "Content-Length: ") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(strings.Split(line, ":")[1]))
			if err != nil {
				log.Println(err)
			}
		}

		if line == "\r\n" || line == "\n" {
			request.WriteString(line)
			break
		}
		request.WriteString(line)
	}

	if contentLength > 0 {
		body := make([]byte, contentLength)
		_, err := io.ReadFull(reader, body)
		if err != nil {
			log.Println(err)
		}

		request.Write(body)
	}

	return request.String()
}

func (h *Handler) HandleRequest(request []string) {
	h = &Handler{
		listener: h.listener,
		conn:     h.conn,
		serveDir: h.serveDir,
		request:  request,
	}

	path := strings.Split(request[0], " ")[1]

	switch strings.Split(request[0], " ")[0] {
	case "GET":
		if path == "/" {
			_, err := h.Root()
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/echo") {
			_, err := h.Echo()
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/user-agent") {
			_, err := h.UserAgent()
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/files") {
			_, err := h.Files()
			if err != nil {
				log.Println(err)
			}
		} else {
			_, err := h.conn.Write([]byte(httpStatus["not found"] + "\r\n"))
			if err != nil {
				log.Println(err)
			}
		}
	case "POST":
		if strings.HasPrefix(path, "/files") {
			_, err := h.NewFile()
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (h *Handler) Root() (int, error) {
	encoding := h.GetEncoding()
	connection, closeConn := h.HandleCloseConnection()

	var root string

	if closeConn {
		root = httpStatus["ok"] + "Content-Length: 0" + encoding + connection + "\r\n"
	} else {
		root = httpStatus["ok"] + "Content-Length: 0" + encoding + connection
	}

	status, err := h.conn.Write([]byte(root))
	if err != nil {
		return status, err
	}

	tcpConn, ok := h.conn.(*net.TCPConn)
	if ok {
		err := tcpConn.SetNoDelay(true)
		if err != nil {
			log.Println(err)
		}
	}

	if closeConn {
		err = h.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return status, err
}

func (h *Handler) Echo() (int, error) {
	body := strings.Split(h.request[0], " ")[1]
	body = strings.ReplaceAll(body, "/echo/", "")

	encoding := h.GetEncoding()
	connection, closeConn := h.HandleCloseConnection()

	var echo string
	var responseBody bytes.Buffer
	var contentLength int

	if strings.Contains(encoding, "gzip") {
		gz := gzip.NewWriter(&responseBody)
		_, err := gz.Write([]byte(body))
		if err != nil {
			log.Println(err)
		}

		err = gz.Close()
		if err != nil {
			log.Println(err)
		}

		contentLength = responseBody.Len()
	} else {
		responseBody.WriteString(body)
		contentLength = utf8.RuneCountInString(body)
	}

	if strings.Contains(encoding, "gzip") {
		echo = fmt.Sprintf(
			"%sContent-Type: text/plain\r\nContent-Length: %d\r\n%s%s%s",
			httpStatus["ok"],
			contentLength,
			encoding,
			connection,
			&responseBody,
		)
	} else if closeConn {
		echo = fmt.Sprintf(
			"%sContent-Type: text/plain\r\nContent-Length: %d\r\n%s\r\n%s",
			httpStatus["ok"],
			contentLength,
			connection,
			&responseBody,
		)
	} else {
		echo = fmt.Sprintf(
			"%sContent-Type: text/plain\r\nContent-Length: %d\r\n%s%s",
			httpStatus["ok"],
			contentLength,
			connection,
			&responseBody,
		)
	}

	status, err := h.conn.Write([]byte(echo))
	if err != nil {
		return status, err
	}

	tcpConn, ok := h.conn.(*net.TCPConn)
	if ok {
		err := tcpConn.SetNoDelay(true)
		if err != nil {
			log.Println(err)
		}
	}

	if closeConn {
		err = h.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return status, err
}

func (h *Handler) UserAgent() (int, error) {
	var body string

	for i := range h.request {
		if strings.Contains(h.request[i], "User-Agent: ") {
			body = strings.ReplaceAll(h.request[i], "User-Agent: ", "")
		}
	}

	var userAgent string

	encoding := h.GetEncoding()
	connection, closeConn := h.HandleCloseConnection()

	if closeConn {
		userAgent = fmt.Sprintf(
			"%sContent-Type: text/plain%sContent-Length: %d\r\n%s\r\n%s",
			httpStatus["ok"],
			encoding,
			utf8.RuneCountInString(body),
			connection,
			body,
		)
	} else {
		userAgent = fmt.Sprintf(
			"%sContent-Type: text/plain%sContent-Length: %d\r\n\r\n%s",
			httpStatus["ok"],
			encoding,
			utf8.RuneCountInString(body),
			body,
		)
	}

	status, err := h.conn.Write([]byte(userAgent))
	if err != nil {
		return status, err
	}

	tcpConn, ok := h.conn.(*net.TCPConn)
	if ok {
		err := tcpConn.SetNoDelay(true)
		if err != nil {
			log.Println(err)
		}
	}

	if closeConn {
		err = h.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return status, err
}

func (h *Handler) Files() (int, error) {
	body := strings.Split(h.request[0], " ")[1]
	body = strings.ReplaceAll(body, "/files/", "")

	var files string

	encoding := h.GetEncoding()
	connection, closeConn := h.HandleCloseConnection()

	fileContent, err := os.ReadFile(h.serveDir + body)
	if err != nil {
		log.Println(err)

		status, err := h.conn.Write([]byte(httpStatus["not found"] + encoding + "\r\n"))
		if err != nil {
			log.Println(err)
		}

		return status, err
	}

	if closeConn {
		files = fmt.Sprintf(
			"%sContent-Type: application/octet-stream\r\nContent-Length: %d\r\n%s%s%s",
			httpStatus["ok"],
			len(fileContent),
			connection,
			encoding,
			string(fileContent[:]),
		)
	} else {
		files = fmt.Sprintf(
			"%sContent-Type: application/octet-stream\r\nContent-Length: %d\r\n%s%s",
			httpStatus["ok"],
			len(fileContent),
			connection,
			string(fileContent[:]),
		)
	}

	status, err := h.conn.Write([]byte(files))
	if err != nil {
		return status, err
	}

	tcpConn, ok := h.conn.(*net.TCPConn)
	if ok {
		err := tcpConn.SetNoDelay(true)
		if err != nil {
			log.Println(err)
		}
	}

	if closeConn {
		err = h.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return status, err
}

func (h *Handler) NewFile() (int, error) {
	log.Println("newfile")
	path := strings.Split(h.request[0], " ")[1]
	fileName := strings.TrimPrefix(path, "/files/")

	var fileContent string
	for i, line := range h.request {
		if line == "" && i+1 < len(h.request) {
			fileContent = strings.TrimSpace(h.request[i+1])
			break
		}
	}

	encoding := h.GetEncoding()
	connection, closeConn := h.HandleCloseConnection()

	file, err := os.Create(h.serveDir + fileName)
	if err != nil {
		log.Println(err)
	}

	fileContentByte := bytes.Trim([]byte(fileContent), "\x00")

	err = os.WriteFile(filepath.Join(h.serveDir, fileName), fileContentByte, 0666)
	if err != nil {
		log.Println(err)
	}

	status, err := h.conn.Write([]byte(httpStatus["created"] + "Content-Length: 0" + encoding + connection + "\r\n"))
	if err != nil {
		return status, err
	}

	err = file.Sync()
	if err != nil {
		log.Println(err)
	}

	err = file.Close()
	if err != nil {
		log.Println(err)
	}

	tcpConn, ok := h.conn.(*net.TCPConn)
	if ok {
		err := tcpConn.SetNoDelay(true)
		if err != nil {
			log.Println(err)
		}
	}

	if closeConn {
		err = h.conn.Close()
		if err != nil {
			log.Println(err)
		}
	}

	return status, err
}
