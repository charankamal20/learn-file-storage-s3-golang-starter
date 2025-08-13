package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const MAXMEMORY = 10 << 20 // In MB

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(MAXMEMORY); err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to retrieve thumbnail file", err)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	parts := strings.Split(contentType, "/")
	if len(parts) != 2 {
		respondWithError(w, http.StatusBadRequest, "Invalid content type format", nil)
		return
	}
	extension := parts[1]

	videoIDStr := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid video ID", err)
		return
	}

	videodata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Video not found", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing JWT token", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid JWT token", err)
		return
	}

	if userID != videodata.UserID {
		respondWithError(w, http.StatusForbidden, "User is not the owner of the video", nil)
		return
	}

	imgData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to read thumbnail", err)
		return
	}
	videoThumbnails[videodata.ID] = thumbnail{
		mediaType: contentType,
		data:      imgData,
	}

	// Save thumbnail to disk
	filename := fmt.Sprintf("%s.%s", videoID.String(), extension)
	filePath := filepath.Join(cfg.assetsRoot, filename)

	outputFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create file on disk", err)
		return
	}
	defer outputFile.Close()

	if _, err := outputFile.Write(imgData); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to write file", err)
		return
	}

	// Update DB
	thumbnailURL := fmt.Sprintf("http://localhost:%s/%s", cfg.port, filePath)
	videodata.ThumbnailURL = &thumbnailURL

	if err := cfg.db.UpdateVideo(videodata); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video in DB", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videodata)
}
