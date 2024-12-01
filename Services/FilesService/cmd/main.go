package main

import (
	"FilesService/internal/handler"
	"FilesService/internal/middleware"
	"FilesService/internal/repository"
	"FilesService/internal/service"
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	mux := http.NewServeMux()
	db, err := openDB("root:1R2o3m4a?@/CloudStorage_FileStorageService?parseTime=true")
	if err != nil {
		print("Error in opening db")
	}
	defer db.Close()

	filesHandler := handler.NewFilesHandler(service.NewFilesService(*repository.NewRepository(db)))

	mux.Handle("/files/", middleware.VerifyAuthMiddleware(filesHandler.GetFileByHash()))
	mux.Handle("/files", middleware.VerifyAuthMiddleware(filesHandler.GetAllFiles()))
	mux.Handle("/add_file/", middleware.VerifyAuthMiddleware(filesHandler.AddFile()))
	mux.Handle("/download/", filesHandler.DownloadFile())
	mux.Handle("/stop-share-status/", middleware.VerifyAuthMiddleware(filesHandler.ChangeShareStatus(false)))
	mux.Handle("/start-share-status/", middleware.VerifyAuthMiddleware(filesHandler.ChangeShareStatus(true)))
	mux.Handle("/delete/", middleware.VerifyAuthMiddleware(filesHandler.DeleteFile()))

	log.Println("Files service starting on port 9996...")
	if err := http.ListenAndServe(":9996", mux); err != nil {
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
