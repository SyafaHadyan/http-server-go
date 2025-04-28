package handler

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"slices"
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
}

func init() {
	supportedEncoding = make([]string, 1)
	supportedEncoding[0] = "gzip"

	httpStatus = make(map[string]string)
	httpStatus["ok"] = "HTTP/1.1 200 OK\r\n"
	httpStatus["not found"] = "HTTP/1.1 404 Not Found\r\n"
	httpStatus["created"] = "HTTP/1.1 201 Created\r\n"
}

func getEncoding(request []string) string {
	for i := range request {
		current := request[i]

		if strings.HasPrefix(current, "Accept-Encoding: ") {
			current := strings.Split(strings.ReplaceAll(current, "Accept-Encoding: ", ""), ", ")
			for j := range current {
				if slices.Contains(supportedEncoding, current[j]) {
					return "Content-Encoding: " + current[j]
				}
			}
		}
	}

	return ""
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
	// reader := bufio.NewReader(h.conn)

	// req, err := http.ReadRequest(reader)
	// if err != nil {
	// 	log.Println(err)
	// }

	// body, err := io.ReadAll(req.Body)
	// if err != nil {
	// 	log.Println(err)
	// }

	// h = &Handler{
	// 	req: req,
	// }

	// log.Println(req)

	// bytes := make([]byte, 0, 8192)
	// temp := make([]byte, 1024)
	// for {
	// 	n, err := h.conn.Read(temp)
	// 	if err == io.EOF {
	// 		break
	// 	}

	// 	bytes = append(bytes, temp[:n]...)
	// }

	// var bytes bytes.Buffer
	// io.Copy(&bytes, h.conn)

	// log.Println(bytes.String())

	// bytes := make([]byte, 8192)
	// request := make([]string, 8)

	// for {
	//	index := 0

	//	_, err := h.conn.Read(bytes)
	//	if err != nil {
	//		break
	//	}

	//	request[index] = string(bytes)
	//	index++
	//}

	// reader := bufio.NewReader(h.conn)
	// var requestBuild strings.Builder

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

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}

		if line == "\r\n" || line == "\n" {
			request.WriteString(line)
			break
		}
		request.WriteString(line)
	}

	return request.String()
}

func (h *Handler) HandleRequest(request []string) {
	if len(request[0]) <= 1 {
		return
	}

	path := strings.Split(request[0], " ")[1]

	switch strings.Split(request[0], " ")[0] {
	case "GET":
		if path == "/" {
			_, err := h.Root(request)
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/echo") {
			_, err := h.Echo(request)
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/user-agent") {
			_, err := h.UserAgent(request)
			if err != nil {
				log.Println(err)
			}
		} else if strings.HasPrefix(path, "/files") {
			_, err := h.Files(request)
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
			_, err := h.NewFile(request)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

func (h *Handler) Root(request []string) (int, error) {
	encoding := getEncoding(request)

	status, err := h.conn.Write([]byte(httpStatus["ok"] + "Content-Length: 0\r\n" + encoding + "\r\n"))
	if err != nil {
		return status, err
	}

	return status, err
}

func (h *Handler) Echo(request []string) (int, error) {
	body := strings.Split(request[0], " ")[1]
	body = strings.ReplaceAll(body, "/echo/", "")

	encoding := getEncoding(request)

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
			"%sContent-Type: text/plain\r\n%s\r\nContent-Length: %d\r\n\r\n%s",
			httpStatus["ok"],
			encoding,
			contentLength,
			&responseBody,
		)
	} else {
		echo = fmt.Sprintf(
			"%sContent-Type: text/plain\r\n%sContent-Length: %d\r\n\r\n%s",
			httpStatus["ok"],
			encoding,
			contentLength,
			&responseBody,
		)
	}

	status, err := h.conn.Write([]byte(echo))
	if err != nil {
		return status, err
	}

	return status, err
}

func (h *Handler) UserAgent(request []string) (int, error) {
	var body string

	for i := range request {
		if strings.Contains(request[i], "User-Agent: ") {
			body = strings.ReplaceAll(request[i], "User-Agent: ", "")
		}
	}

	encoding := getEncoding(request)

	userAgent := fmt.Sprintf(
		"%s%sContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		httpStatus["ok"],
		encoding,
		utf8.RuneCountInString(body),
		body,
	)

	status, err := h.conn.Write([]byte(userAgent))
	if err != nil {
		return status, err
	}

	// err = h.conn.Close()
	if err != nil {
		log.Println(err)
	}

	return status, err
}

func (h *Handler) Files(request []string) (int, error) {
	body := strings.Split(request[0], " ")[1]
	body = strings.ReplaceAll(body, "/files/", "")

	encoding := getEncoding(request)

	fileContent, err := os.ReadFile(h.serveDir + body)
	if err != nil {
		log.Println(err)

		status, err := h.conn.Write([]byte(httpStatus["not found"] + encoding + "\r\n"))
		if err != nil {
			log.Println(err)
		}

		return status, err
	}

	files := fmt.Sprintf(
		"%s%sContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s",
		httpStatus["ok"],
		encoding,
		len(fileContent),
		string(fileContent[:]),
	)

	status, err := h.conn.Write([]byte(files))
	if err != nil {
		return status, err
	}

	// err = h.conn.Close()
	if err != nil {
		log.Println(err)
	}

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

	encoding := getEncoding(request)

	file, err := os.Create(h.serveDir + fileName)
	if err != nil {
		log.Println(err)
	}

	fileContentByte := bytes.Trim([]byte(fileContent), "\x00")

	err = os.WriteFile(filepath.Join(h.serveDir, fileName), fileContentByte, 0666)
	if err != nil {
		log.Println(err)
	}

	status, err := h.conn.Write([]byte(httpStatus["created"] + encoding + "\r\n"))
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

	// err = h.conn.Close()
	if err != nil {
		log.Println(err)
	}

	return status, err
}
