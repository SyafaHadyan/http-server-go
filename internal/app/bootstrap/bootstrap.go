package bootstrap

import (
	"github.com/SyafaHadyan/http-server-go/internal/app/handler"
)

func Start() error {
	return handler.NewHandler()
}
