package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
	"video_solicitation_microservice/internal/core/use_case"
)

type VideoHandler struct {
	createVideo     *use_case.CreateVideo
	getDownloadLink *use_case.GetDownloadLink
}

func NewVideoHandler(createVideo *use_case.CreateVideo, getDownloadLink *use_case.GetDownloadLink) *VideoHandler {
	return &VideoHandler{
		createVideo:     createVideo,
		getDownloadLink: getDownloadLink,
	}
}

func (h *VideoHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/videos", h.CreateVideo)
	r.GET("/videos/:video_id/download", h.GetDownloadLink)
}

func (h *VideoHandler) CreateVideo(c *gin.Context) {
	var input dto.CreateVideoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	output, err := h.createVideo.Execute(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, exception.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create video solicitation"})
		return
	}

	c.JSON(http.StatusCreated, output)
}

func (h *VideoHandler) GetDownloadLink(c *gin.Context) {
	videoID := c.Param("video_id")

	output, err := h.getDownloadLink.Execute(c.Request.Context(), videoID)
	if err != nil {
		if errors.Is(err, exception.ErrVideoNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, exception.ErrVideoNotCompleted) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, exception.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get download link"})
		return
	}

	c.JSON(http.StatusOK, output)
}
