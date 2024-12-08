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
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	db, err := openDB("root:1R2o3m4a?@/CloudStorage_FileStorageService?parseTime=true")
	if err != nil {
		print("Error in opening db")
	}
	defer db.Close()

	filesHandler := handler.NewFilesHandler(service.NewFilesService(*repository.NewFileRepository(db)), service.NewFolderService(*repository.NewFolderRepository(db)))

	mux.Handle("/files/", filesHandler.GetFileByHash())
	mux.Handle("/folders/", middleware.VerifyAuthMiddleware(filesHandler.GetFolderEntities()))
	mux.Handle("/add_file/", middleware.VerifyAuthMiddleware(filesHandler.AddFile()))
	mux.Handle("/download/", filesHandler.DownloadFile())
	mux.Handle("/stop-share-status/", middleware.VerifyAuthMiddleware(filesHandler.ChangeShareStatus(false)))
	mux.Handle("/start-share-status/", middleware.VerifyAuthMiddleware(filesHandler.ChangeShareStatus(true)))
	mux.Handle("/delete-file/", middleware.VerifyAuthMiddleware(filesHandler.DeleteFile()))
	mux.Handle("/delete-folder/", middleware.VerifyAuthMiddleware(filesHandler.DeleteFolder()))
	mux.Handle("/create_folder/", middleware.VerifyAuthMiddleware(filesHandler.CreateFolder()))

	services := []string{"http://localhost:9997", "http://localhost:9998", "http://localhost:9999", "http://localhost:9996", "http://localhost:9995"}
	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   services,                                 // Разрешаем только домен фронтенда
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"}, // Разрешаем методы
		AllowedHeaders:   []string{"Content-Type"},                 // Разрешаем заголовок Content-Type
		AllowCredentials: true,                                     // Разрешаем куки
	})

	log.Println("API Gateway starting on port 9996...")
	if err := http.ListenAndServe(":9996", corsHandler.Handler(mux)); err != nil {
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
