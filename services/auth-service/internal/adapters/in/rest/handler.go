package rest

import (
	"net/http"

	"github.com/fiapx/auth-service/internal/application"
	"github.com/fiapx/auth-service/internal/domain"
	"github.com/fiapx/auth-service/internal/metrics"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	auth *application.AuthUseCase
}

func NewHandler(auth *application.AuthUseCase) *Handler {
	return &Handler{auth: auth}
}

func (h *Handler) Router() *gin.Engine {
	r := gin.Default()
	r.Use(metrics.Middleware())
	r.GET("/health", h.health)
	r.GET("/metrics", metrics.Handler())
	r.POST("/register", h.register)
	r.POST("/login", h.login)
	return r
}

type credentialsRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "auth-service"})
}

func (h *Handler) register(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	user, err := h.auth.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		switch err {
		case domain.ErrUserAlreadyExists:
			c.JSON(http.StatusConflict, gin.H{"error": "email already registered"})
		case domain.ErrInvalidInput:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid email or password (min 6 chars)"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}

	metrics.RegistrationsInc()
	c.JSON(http.StatusCreated, gin.H{"id": user.ID, "email": user.Email})
}

func (h *Handler) login(c *gin.Context) {
	var req credentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if err == domain.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	metrics.LoginsInc()
	c.JSON(http.StatusOK, gin.H{"token": token})
}
