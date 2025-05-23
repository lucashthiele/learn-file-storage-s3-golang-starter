package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

const MAX_MEMORY int64 = 10 << 20 // 10485760 bytes - 10MB
const FORM_FILE_KEY string = "thumbnail"

func getFilename(contentType string, videoID uuid.UUID) (string, string) {
	fileExtension := strings.Split(contentType, "/")[1]
	filename := fmt.Sprintf("%s.%s", videoID.String(), fileExtension)
	return fileExtension, filename
}

func createFile(cfg *apiConfig, filename string) (*os.File, error) {
	imagePath := filepath.Join(cfg.assetsRoot, filename)
	imageFile, err := os.Create(imagePath)
	return imageFile, err
}

func getThumbnailURL(port, videoID, fileExtension string) *string {
	thumbnailURL := fmt.Sprintf("http://localhost:%s/assets/%s.%s", port, videoID, fileExtension)
	return &thumbnailURL
}

func validateContentType(contentType string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", err
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		return "", fmt.Errorf("media type %s not supported", mediaType)
	}

	return mediaType, nil
}

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

	reqFile, headers, err := req.FormFile(FORM_FILE_KEY)
	if err != nil {
		respondWithError(resp, http.StatusBadRequest, "Couldn't get file", err)
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

	contentType := headers.Header.Get("Content-Type")
	mediaType, err := validateContentType(contentType)
	if err != nil {
		respondWithError(resp, http.StatusUnsupportedMediaType, err.Error(), err)
		return
	}

	fileExtension, filename := getFilename(mediaType, videoID)

	imageFile, err := createFile(cfg, filename)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Couldn't create image file", err)
		return
	}

	_, err = io.Copy(imageFile, reqFile)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Couldn't create image file", err)
		return
	}

	video.ThumbnailURL = getThumbnailURL(cfg.port, video.ID.String(), fileExtension)

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(resp, http.StatusInternalServerError, "Couldn't update video in db", err)
		return
	}

	respondWithJSON(resp, http.StatusOK, video)
}
