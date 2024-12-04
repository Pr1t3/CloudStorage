package main

import (
	"FrontendService/internal/handler"
	"FrontendService/internal/middleware"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/login/", middleware.VerifyNotAuthMiddleware(handler.LoginHandler()))
	mux.Handle("/register/", middleware.VerifyNotAuthMiddleware(handler.RegisterHandler()))
	mux.Handle("/files", middleware.VerifyAuthMiddleware(handler.ShowAllFiles()))
	mux.Handle("/add_file/", middleware.VerifyAuthMiddleware(handler.AddFileHandler()))
	mux.Handle("/files/", middleware.VerifyAuthMiddleware(handler.ShowFile()))
	mux.Handle("/", middleware.VerifyNotAuthMiddleware(handler.LoginHandler()))
	mux.Handle("/profile/", middleware.VerifyAuthMiddleware(handler.ShowProfile()))

	log.Println("Frontend Service starting on port 9997...")
	if err := http.ListenAndServe(":9997", mux); err != nil {
		log.Fatalf("Failed to start Frontend Service: %v", err)
	}
}
