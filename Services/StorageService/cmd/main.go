package main

import (
	"StorageService/internal/handler"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	storageHandler := handler.NewStorageHandler()

	mux.Handle("/upload/", storageHandler.UploadFile())
	mux.Handle("/download", storageHandler.DownloadFile())
	mux.Handle("/delete", storageHandler.DeleteFile())

	log.Println("Storage Service starting on port 9995...")
	if err := http.ListenAndServe(":9995", mux); err != nil {
		log.Fatalf("Failed to start Storage Service: %v", err)
	}
}
