package bootstrap

import (
	"github.com/SyafaHadyan/http-server-go/internal/app/handler"
)

func Start() (int, error) {
	return handler.NewHandler()
}
