package bootstrap

import (
	"github.com/SyafaHadyan/http-server-go/internal/app/handler"
)

func Start() {
	handler.NewHandler()
}
