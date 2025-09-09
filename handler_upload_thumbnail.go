package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	const maxMemory = 10 << 20

	if err := r.ParseMultipartForm(maxMemory); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse request", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, 400, "Cannot read file", err)
		return
	}

	videoRecord, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 400, "Failed to get video record", err)
		return
	}
	if videoRecord.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorised", err)
		return
	}

	encodedFile := base64.StdEncoding.EncodeToString(fileData)

	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, encodedFile)

	videoRecord.ThumbnailURL = &dataURL

	if err := cfg.db.UpdateVideo(videoRecord); err != nil {
		respondWithError(w, 400, "Failed to write to database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoRecord)
}
