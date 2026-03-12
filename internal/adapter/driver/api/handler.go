package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"video_solicitation_microservice/internal/core/dto"
	"video_solicitation_microservice/internal/core/exception"
	"video_solicitation_microservice/internal/core/use_case"
)

type VideoHandler struct {
	createVideo       *use_case.CreateVideo
	getDownloadLink   *use_case.GetDownloadLink
	getVideosByUser   *use_case.GetVideosByUser
	updateVideoStatus *use_case.UpdateVideoStatus
	updateChunkStatus *use_case.UpdateChunkStatus
}

func NewVideoHandler(createVideo *use_case.CreateVideo, getDownloadLink *use_case.GetDownloadLink, getVideosByUser *use_case.GetVideosByUser, updateVideoStatus *use_case.UpdateVideoStatus, updateChunkStatus *use_case.UpdateChunkStatus) *VideoHandler {
	return &VideoHandler{
		createVideo:       createVideo,
		getDownloadLink:   getDownloadLink,
		getVideosByUser:   getVideosByUser,
		updateVideoStatus: updateVideoStatus,
		updateChunkStatus: updateChunkStatus,
	}
}

func (h *VideoHandler) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/videos", h.CreateVideo)
	r.GET("/videos/:video_id/download", h.GetDownloadLink)
	r.GET("/videos/user/:user_id", h.GetVideosByUser)
	r.PATCH("/videos/:video_id/status", h.UpdateVideoStatus)
	r.PATCH("/videos/:video_id/chunks/:part_number/status", h.UpdateChunkStatus)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

func (h *VideoHandler) GetVideosByUser(c *gin.Context) {
	userID := c.Param("user_id")
	videos, err := h.getVideosByUser.Execute(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, videos)
}

func (h *VideoHandler) UpdateVideoStatus(c *gin.Context) {
	videoID := c.Param("video_id")
	var input dto.UpdateVideoStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	input.VideoID = videoID
	if err := h.updateVideoStatus.Execute(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "video status updated"})
}

func (h *VideoHandler) UpdateChunkStatus(c *gin.Context) {
	videoID := c.Param("video_id")
	partNumber := c.Param("part_number")
	var input dto.UpdateChunkStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}
	input.VideoID = videoID
	pn, _ := strconv.Atoi(partNumber)
	input.ChunkPart = pn
	if err := h.updateChunkStatus.Execute(c.Request.Context(), input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "chunk status updated"})
}
