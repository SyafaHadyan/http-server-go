package bootstrap

import (
	"fmt"

	"github.com/SyafaHadyan/http-server-go/internal/app/handler"
	"github.com/gofiber/fiber/v2"
)

func Start() error {
	app := fiber.New()

	routerGroup := app.Group("/")

	handler.NewHandler(routerGroup)

	return app.Listen(fmt.Sprintf(":%d", 4221))
}
