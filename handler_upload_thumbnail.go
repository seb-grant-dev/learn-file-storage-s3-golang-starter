package main

import (
	"fmt"
	"net/http"
	"io"
	"os"
	"path/filepath"
	"errors"
	"mime"

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
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read thumbnail field", err)
		return
	}
	defer file.Close()

	fileType := header.Header["Content-Type"]

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail image data", err)
		return
	}

	videoMeta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to locate video", err)
		return
	}

	if videoMeta.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Video does not belong to current user",err)
		return
	}

	mediatype, _, err := mime.ParseMediaType(fileType[0])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse thumbnail upload",err)
		return
	}

	var fileExtn string
	switch mediatype {
	case "image/png":
		fileExtn = "png"
		break
	case "image/jpg":
	case "image/jpeg":
		fileExtn = "jpg"
	default:
		err = errors.New("Incorrect thumbnail type uploaded")
		respondWithError(w, http.StatusBadRequest,"Thumbnails must be .png or .jpg/.jpeg" , err)
		return
	}

	thumbnailFileName := fmt.Sprintf("%s.%s",videoID.String(),fileExtn)
	thumbnailAssetPath := filepath.Join(cfg.assetsRoot,thumbnailFileName)
	thumbnailAssetUrl := fmt.Sprintf("http://localhost:%s/%s",cfg.port,thumbnailAssetPath)
	fmt.Println("Storing thubnail file at: ",thumbnailAssetPath)

	thumbFile, err := os.Create(thumbnailAssetPath)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save thumbnail to file system",err)
		return
	}

	copiedBytes, err := io.Copy(thumbFile,file)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error writing thumbnail to file system",err)
		return
	}

	fmt.Printf("Wrote %d bytes to disk\n",copiedBytes)

	thumbnailUrl := thumbnailAssetUrl
	videoMeta.ThumbnailURL = &thumbnailUrl
	err = cfg.db.UpdateVideo(videoMeta)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to save video data", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoMeta)
}
