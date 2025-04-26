package main

import (
	"os"

	"github.com/SyafaHadyan/http-server-go/internal/app/bootstrap"
)

func main() {
	bootstrap.Start(os.Args)
}
