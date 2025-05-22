package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const MAX_MEMORY int64 = 10 << 20 // 10485760 bytes - 10MB
const FORM_FILE_KEY string = "thumbnail"

func (cfg *apiConfig) handlerUploadThumbnail(resp http.ResponseWriter, req *http.Request) {
	videoIDString := req.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(resp, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(resp, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	err = req.ParseMultipartForm(MAX_MEMORY)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, "Couldn't parse multipart form", err)
		return
	}

	file, headers, err := req.FormFile(FORM_FILE_KEY)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, "Couldn't get file", err)
		return
	}

	contentType := headers.Header.Get("Content-Type")

	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, "Couldn't read image", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Couldn't get video from db", err)
		return
	}

	if video.UserID != userID {
		respondWithError(resp, http.StatusUnauthorized, "User is not the owner of the video", err)
		return
	}

	imageBase64 := base64.StdEncoding.EncodeToString(imageData)

	thumbnailURL := fmt.Sprintf("data:%s;base64,%s", contentType, imageBase64)

	video.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Couldn't update video in db", err)
		return
	}

	respondWithJSON(resp, http.StatusOK, video)
}
