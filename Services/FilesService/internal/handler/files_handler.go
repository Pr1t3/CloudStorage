package handler

import (
	"FilesService/internal/service"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Claims struct {
	Email  string
	UserId int
	exp    time.Time
	iat    time.Time
}

func ProxyRequest(r *http.Request, target string) ([]byte, error) {
	proxyReq, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		return nil, err
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}
	for _, cookie := range r.Cookies() {
		proxyReq.AddCookie(cookie)
	}

	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))
	return body, nil
}

type FilesHandler struct {
	filesService *service.FilesService
}

func NewFilesHandler(filesService *service.FilesService) *FilesHandler {
	return &FilesHandler{filesService: filesService}
}

func (f *FilesHandler) GetAllFiles() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		userId := claims.UserId
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		files, err := f.filesService.GetAllFiles(userId)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		jsonData, err := json.Marshal(files)
		if err != nil {
			http.Error(w, "Failed to marshal files", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))
		w.Write(jsonData)
	})
}

func (f *FilesHandler) GetFileByHash() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path

		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "Некорректный путь", http.StatusBadRequest)
			return
		}

		hash := parts[2]
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		userId := claims.UserId

		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		if file.User_id != userId {
			log.Println(err)
			http.Error(w, "404 Page Not Found", http.StatusNotFound)
			return
		}
		jsonData, err := json.Marshal(file)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to marshal files", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(jsonData)))
		w.Write(jsonData)
	})
}

func (f *FilesHandler) AddFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		userId := claims.UserId

		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при разборе формы", http.StatusBadRequest)
			return
		}

		file, fileHeader, err := r.FormFile("fileToUpload")
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при получении файла", http.StatusBadRequest)
			return
		}
		defer file.Close()
		fileName := fileHeader.Filename

		fileType := fileHeader.Header.Get("Content-Type")

		dir := "./uploads/" + strconv.Itoa(userId)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err = os.Mkdir(dir, os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		filePath := filepath.Join(dir, fileName)
		dst, err := os.Create(filePath)
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при сохранении файла", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при копировании файла", http.StatusInternalServerError)
			return
		}
		err = f.filesService.AddFile(userId, fileName, filePath, fileType)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (f *FilesHandler) DownloadFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path

		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "Некорректный путь", http.StatusBadRequest)
			return
		}

		hash := parts[2]
		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !file.ShareStatus {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

			respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

			var claims Claims

			if err := json.Unmarshal(respBody, &claims); err != nil {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}

			if claims.UserId != file.User_id {
				log.Println(err)
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
		}
		openedFile, err := os.Open(file.FilePath)

		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer openedFile.Close()

		fileInfo, err := openedFile.Stat()
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name()))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

		http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), openedFile)
	})
}

func (f *FilesHandler) GetFileData() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path

		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "Некорректный путь", http.StatusBadRequest)
			return
		}

		hash := parts[2]
		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !file.ShareStatus {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
				return
			}
			defer r.Body.Close()
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

			respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

			var claims Claims

			if err := json.Unmarshal(respBody, &claims); err != nil {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
			if claims.UserId != file.User_id {
				log.Println(err)
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
		}
		if strings.HasPrefix(file.FileType, "image") {
			openedFile, err := os.Open(file.FilePath)

			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			defer openedFile.Close()

			fileInfo, err := openedFile.Stat()
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileInfo.Name()))
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

			http.ServeContent(w, r, fileInfo.Name(), fileInfo.ModTime(), openedFile)
		} else if strings.HasPrefix(file.FileType, "text") {
			fileData, err := os.ReadFile(file.FilePath)
			if err != nil {
				http.Error(w, "File not found", http.StatusNotFound)
				return
			}

			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(fileData)))

			w.Write(fileData)
		}
	})
}

func (f *FilesHandler) ChangeShareStatus(status bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path

		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "Некорректный путь", http.StatusBadRequest)
			return
		}

		hash := parts[2]

		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if claims.UserId != file.User_id {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		err = f.filesService.ChangeShareStatus(hash, status)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func (f *FilesHandler) DeleteFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path

		parts := strings.Split(path, "/")
		if len(parts) < 3 {
			http.Error(w, "Некорректный путь", http.StatusBadRequest)
			return
		}

		hash := parts[2]
		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, err := ProxyRequest(r, "http://localhost:9999/get-claims/")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		if claims.UserId != file.User_id {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}
		err = os.Remove(file.FilePath)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		err = f.filesService.DeleteFile(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
	})
}
