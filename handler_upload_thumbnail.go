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

	// Bit shifting is a way to multiply by powers of 2. 10 << 20 is the
	// same as 10 * 1024 * 1024, which is 10MB.
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()
	mediaType := header.Header.Get("Content-Type")

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to fetch video from DB", err)
		return
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized!", err)
		return
	}

	// encodedData := base64.StdEncoding.EncodeToString(data)
	// dataUrl := fmt.Sprintf("data:%s;base64,%s", mediaType, encodedData)
	// video.ThumbnailURL = &dataUrl
	// err = cfg.db.UpdateVideo(video)

	fileExtension := strings.Split(mediaType, "/")[1]
	assetFilename := fmt.Sprintf("%s.%s", video.ID, fileExtension)
	assetpath := filepath.Join(cfg.assetsRoot, assetFilename)

	asset, err := os.Create(assetpath)
	if err != nil {
        os.Remove(assetpath)
		respondWithError(w, http.StatusBadRequest, "Unable to create asset!", err)
		return
	}
    defer asset.Close()

	if _, err = io.Copy(asset, file); err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save asset!", err)
		return
	}

	thumbnailUrl := fmt.Sprintf("http://localhost:%s/%s", cfg.port, assetpath)
	video.ThumbnailURL = &thumbnailUrl

	if err = cfg.db.UpdateVideo(video); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
