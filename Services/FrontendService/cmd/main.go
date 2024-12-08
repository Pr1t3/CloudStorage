package main

import (
	"FrontendService/internal/handler"
	"FrontendService/internal/middleware"
	"log"
	"net/http"

	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/login/", middleware.VerifyNotAuthMiddleware(handler.LoginHandler()))
	mux.Handle("/register/", middleware.VerifyNotAuthMiddleware(handler.RegisterHandler()))
	mux.Handle("/folders/", middleware.VerifyAuthMiddleware(handler.ShowFolderEntities()))
	// mux.Handle("/add_file/", middleware.VerifyAuthMiddleware(handler.AddFileHandler()))
	mux.Handle("/files/", middleware.VerifyAuthMiddleware(handler.ShowFile()))
	mux.Handle("/", middleware.VerifyNotAuthMiddleware(handler.LoginHandler()))
	mux.Handle("/profile/", middleware.VerifyAuthMiddleware(handler.ShowProfile()))

	services := []string{"http://localhost:9997", "http://localhost:9998", "http://localhost:9999", "http://localhost:9996", "http://localhost:9995"}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   services,                                 // Разрешаем только домен фронтенда
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"}, // Разрешаем методы
		AllowedHeaders:   []string{"Content-Type"},                 // Разрешаем заголовок Content-Type
		AllowCredentials: true,                                     // Разрешаем куки
	})

	log.Println("API Gateway starting on port 9997...")
	if err := http.ListenAndServe(":9997", corsHandler.Handler(mux)); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
