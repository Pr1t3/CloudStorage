package main

import (
	"ApiGateway/internal/handler"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.Handle("/login/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/files"))
	mux.Handle("/register/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/files"))
	mux.Handle("/logout/", handler.ProxyHandlerRedirect("http://localhost:9999", "http://localhost:9997/login/"))
	mux.Handle("/add_file/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/files"))
	mux.Handle("/download/", handler.ProxyHandler("http://localhost:9996"))
	mux.Handle("/delete/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/files"))
	mux.Handle("/start-share-status/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/files"))
	mux.Handle("/stop-share-status/", handler.ProxyHandlerRedirect("http://localhost:9996", "http://localhost:9997/files"))

	log.Println("API Gateway starting on port 9998...")
	if err := http.ListenAndServe(":9998", mux); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}
