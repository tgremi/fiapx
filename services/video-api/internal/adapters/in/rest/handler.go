package rest

import (
	"io"
	"net/http"
	"time"

	"github.com/fiapx/video-api/internal/application"
	"github.com/fiapx/video-api/internal/domain"
	"github.com/fiapx/video-api/internal/metrics"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	videos    *application.VideoUseCase
	validator domain.TokenValidator
}

func NewHandler(videos *application.VideoUseCase, validator domain.TokenValidator) *Handler {
	return &Handler{videos: videos, validator: validator}
}

func (h *Handler) Router() *gin.Engine {
	r := gin.Default()
	r.MaxMultipartMemory = 100 << 20

	r.Use(metrics.Middleware())
	r.GET("/health", h.health)
	r.GET("/metrics", metrics.Handler())

	api := r.Group("")
	api.Use(authMiddleware(h.validator))
	{
		api.POST("/videos", h.upload)
		api.GET("/videos", h.list)
		api.GET("/videos/:id", h.get)
		api.GET("/videos/:id/download", h.download)
	}

	return r
}

type videoResponse struct {
	ID        string             `json:"id"`
	Status    domain.VideoStatus `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
}

func (h *Handler) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "video-api"})
}

func (h *Handler) upload(c *gin.Context) {
	userID := c.GetString("userID")

	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campo 'video' (multipart) é obrigatório"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao ler arquivo"})
		return
	}
	defer func() { _ = f.Close() }()

	v, err := h.videos.Upload(c.Request.Context(), userID, file.Filename, f, file.Size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao processar upload"})
		return
	}

	metrics.VideosUploadedInc()
	c.JSON(http.StatusAccepted, videoResponse{ID: v.ID, Status: v.Status, CreatedAt: v.CreatedAt})
}

func (h *Handler) list(c *gin.Context) {
	userID := c.GetString("userID")

	vids, err := h.videos.ListByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "erro ao listar vídeos"})
		return
	}

	resp := make([]videoResponse, 0, len(vids))
	for _, v := range vids {
		resp = append(resp, videoResponse{ID: v.ID, Status: v.Status, CreatedAt: v.CreatedAt})
	}

	c.JSON(http.StatusOK, gin.H{"videos": resp, "total": len(resp)})
}

func (h *Handler) get(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	v, err := h.videos.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vídeo não encontrado"})
		return
	}

	c.JSON(http.StatusOK, videoResponse{ID: v.ID, Status: v.Status, CreatedAt: v.CreatedAt})
}

func (h *Handler) download(c *gin.Context) {
	userID := c.GetString("userID")
	id := c.Param("id")

	r, filename, err := h.videos.Download(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vídeo não encontrado ou ainda não processado"})
		return
	}
	defer func() { _ = r.Close() }()

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/zip")
	c.Status(http.StatusOK)
	metrics.VideosDownloadedInc()
	_, _ = io.Copy(c.Writer, r)
}
