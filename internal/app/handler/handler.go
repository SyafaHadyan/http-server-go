package handler

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

type Handler struct{}

func NewHandler(routerGroup fiber.Router) {
	handler := Handler{}

	routerGroup = routerGroup.Group("/")

	routerGroup.Get("/", handler.Root)
}

func (h *Handler) Root(ctx *fiber.Ctx) error {
	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"message": "success",
	})
}
