package handlers

import (
	"diploma-back/internal/config"
	"diploma-back/internal/services"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	handler  *gin.Engine
	Services *services.Service
	Config   *config.Config
}

func NewHandler(services *services.Service, config *config.Config, engine *gin.Engine) *Handler {
	return &Handler{
		handler:  engine,
		Services: services,
		Config:   config,
	}
}

func (h *Handler) InitRoutes() {
	h.InitAuthRoutes()
	h.InitProcessingRoutes()
	h.InitAdminRoutes()
}
