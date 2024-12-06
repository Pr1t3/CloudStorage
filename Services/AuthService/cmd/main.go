package main

import (
	"AuthService/internal/handler"
	"AuthService/internal/middleware"
	"AuthService/internal/repository"
	"AuthService/internal/service"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	db, err := openDB("root:1R2o3m4a?@/CloudStorage_AuthService?parseTime=true")
	if err != nil {
		print("Error in opening db")
	}
	defer db.Close()

	authHandler := handler.NewAuthHandler(service.NewAuthService(*repository.NewRepository(db)))

	mux.Handle("/login/", authHandler.Login())
	mux.Handle("/register/", authHandler.Register())
	mux.Handle("/validate/", authHandler.Validate())
	mux.Handle("/get-claims/", middleware.VerifyAuthMiddleware(authHandler.GetClaims()))
	mux.Handle("/logout/", middleware.VerifyAuthMiddleware(authHandler.Logout()))
	mux.Handle("/get-profile-photo", middleware.VerifyAuthMiddleware(authHandler.GetProfilePhoto()))
	mux.Handle("/change-password", middleware.VerifyAuthMiddleware(authHandler.ChangePassword()))
	mux.Handle("/upload-photo", middleware.VerifyAuthMiddleware(authHandler.UploadPhoto()))

	services := []string{"http://localhost:9997", "http://localhost:9998", "http://localhost:9999", "http://localhost:9996", "http://localhost:9995"}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   services,                                 // Разрешаем только домен фронтенда
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"}, // Разрешаем методы
		AllowedHeaders:   []string{"Content-Type"},                 // Разрешаем заголовок Content-Type
		AllowCredentials: true,                                     // Разрешаем куки
	})

	log.Println("API Gateway starting on port 9999...")
	if err := http.ListenAndServe(":9999", corsHandler.Handler(mux)); err != nil {
		log.Fatalf("Failed to start API Gateway: %v", err)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
