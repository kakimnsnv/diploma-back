package handlers

import (
	"diploma-back/internal/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) InitAdminRoutes() {
	// CORS middleware
	h.handler.Use(middleware.CORSMiddleware())

	admin := h.handler.Group("/api/admin")
	admin.Use(middleware.AuthMiddleware(&h.Config.JWT))
	admin.Use(middleware.RoleMiddleware("admin"))
	{
		admin.GET("/users", h.GetUsersWithJobs)
	}
}

func (h *Handler) GetUsersWithJobs(c *gin.Context) {
	users, err := h.Services.AdminService.GetUsersWithJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}
