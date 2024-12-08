package main

import (
	"StorageService/internal/handler"
	"log"
	"net/http"

	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()

	storageHandler := handler.NewStorageHandler()

	mux.Handle("/upload/", storageHandler.UploadFile())
	mux.Handle("/download", storageHandler.DownloadFile())
	mux.Handle("/delete-file", storageHandler.DeleteFile())
	mux.Handle("/create_folder/", storageHandler.CreateFolder())
	mux.Handle("/delete-folder", storageHandler.DeleteFolder())

	services := []string{"http://localhost:9997", "http://localhost:9998", "http://localhost:9999", "http://localhost:9996", "http://localhost:9995"}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   services,                                 // Разрешаем только домен фронтенда
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"}, // Разрешаем методы
		AllowedHeaders:   []string{"Content-Type"},                 // Разрешаем заголовок Content-Type
		AllowCredentials: true,                                     // Разрешаем куки
	})

	log.Println("API Gateway starting on port 9995...")
	if err := http.ListenAndServe(":9995", corsHandler.Handler(mux)); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
