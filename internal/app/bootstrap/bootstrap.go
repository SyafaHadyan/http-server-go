package bootstrap

import (
	"github.com/SyafaHadyan/http-server-go/internal/app/handler"
)

func Start(args []string) {
	serveDir := "/"

	if len(args) == 3 && args[1] == "--directory" {
		serveDir = args[2]
	}

	handler.NewHandler(serveDir)
}
