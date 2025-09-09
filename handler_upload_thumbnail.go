package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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

	mediaTypeHeader := header.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(mediaTypeHeader)
	if err != nil {
		respondWithError(w, 400, "Failed to get file type", err)
		return
	}
	if mt != "image/jpeg" && mt != "image/png" {
		respondWithError(w, 400, "Incorrect file type", nil)
		return
	}
	exts, err := mime.ExtensionsByType(mt)
	if err != nil {
		respondWithError(w, 400, "Failed to get file type", err)
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

	unique := []byte{}
	rand.Read(unique)
	uniqueString := base64.RawURLEncoding.EncodeToString(unique)

	thumbnailFilepath := filepath.Join(cfg.assetsRoot, uniqueString+exts[0])

	thumbnailFile, err := os.Create(thumbnailFilepath)
	if err != nil {
		respondWithError(w, 500, "Failed to create file", err)
		return
	}
	if _, err := io.Copy(thumbnailFile, file); err != nil {
		respondWithError(w, 500, "Failed to copy thumbnail", err)
		return
	}

	dataURL := fmt.Sprintf("http://localhost:%s/assets/%s", cfg.port, uniqueString+exts[0])

	videoRecord.ThumbnailURL = &dataURL

	if err := cfg.db.UpdateVideo(videoRecord); err != nil {
		respondWithError(w, 400, "Failed to write to database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoRecord)
}
