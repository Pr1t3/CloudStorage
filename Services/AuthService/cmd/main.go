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

	log.Println("Auth service starting on port 9999...")
	if err := http.ListenAndServe(":9999", mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
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
