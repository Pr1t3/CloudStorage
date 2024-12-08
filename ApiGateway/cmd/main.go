package main

import (
	"ApiGateway/internal/handler"
	"log"
	"net/http"

	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/login/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/folders"))
	mux.Handle("/register/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/folders"))
	mux.Handle("/logout/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/login/"))
	mux.Handle("/add_file/", handler.ProxyHandler("http://localhost:9996"))
	mux.Handle("/download/", handler.ProxyHandler("http://localhost:9996"))
	mux.Handle("/delete-file/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/folders"))
	mux.Handle("/delete-folder/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/folders"))
	mux.Handle("/start-share-status/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/folders"))
	mux.Handle("/stop-share-status/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/folders"))
	mux.Handle("/change-password", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/profile/"))
	mux.Handle("/upload-photo", handler.ProxyHandler("http://localhost:9999"))
	mux.Handle("/create_folder/", handler.ProxyHandler("http://localhost:9996"))

	services := []string{"http://localhost:9997", "http://localhost:9998", "http://localhost:9999", "http://localhost:9996", "http://localhost:9995"}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   services,                                 // Разрешаем только домен фронтенда
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"}, // Разрешаем методы
		AllowedHeaders:   []string{"Content-Type", "Hash"},         // Разрешаем заголовок Content-Type
		AllowCredentials: true,                                     // Разрешаем куки
	})

	log.Println("API Gateway starting on port 9998...")
	if err := http.ListenAndServe(":9998", corsHandler.Handler(mux)); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
