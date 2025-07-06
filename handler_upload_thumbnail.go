package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const MAXMEMORY = 10 << 20 // In MB

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(MAXMEMORY)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not parse form data", err)
		return
	}

	file, fileHeader, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get file", err)
		return
	}

	fileformat := fileHeader.Header.Get("Content-Type")

	img, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not read file", err)
		return
	}

	id, err := uuid.Parse(r.PathValue("videoID"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get id", err)
		return
	}

	videodata, err := cfg.db.GetVideo(id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not get video", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	if userID != videodata.UserID {
		respondWithError(w, http.StatusUnauthorized, "Not the owner", err)
		return
	}

	var tn thumbnail
	tn.mediaType = fileformat
	tn.data = img

	videoThumbnails[videodata.ID] = tn

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videodata.ID.String())
	fmt.Println("THUMBNAIL: ", thumbnailUrl)
	videodata.ThumbnailURL = &thumbnailUrl

	err = cfg.db.UpdateVideo(videodata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not update video in db", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videodata)
}
