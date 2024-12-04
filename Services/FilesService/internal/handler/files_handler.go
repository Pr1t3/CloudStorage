package handler

import (
	"FilesService/internal/service"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type Claims struct {
	Email  string
	UserId int
	exp    time.Time
	iat    time.Time
}

func (f *FilesHandler) ProxyRequest(r *http.Request, target string, reqBody io.Reader, method string) ([]byte, *http.Header, error) {
	proxyReq, err := http.NewRequest(method, target, reqBody)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, f.forbiddenError
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, &resp.Header, err
}

type FilesHandler struct {
	filesService   *service.FilesService
	forbiddenError error
}

func NewFilesHandler(filesService *service.FilesService) *FilesHandler {
	return &FilesHandler{filesService: filesService, forbiddenError: errors.New("Status Forbidden")}
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}

		if !file.ShareStatus {
			if err := json.Unmarshal(respBody, &claims); err != nil {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}

			userId := claims.UserId

			if file.User_id != userId {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9995/upload/", r.Body, http.MethodPost)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var responseData struct {
			UserId   int
			FileName string
			FilePath string
			FileType string
		}

		if err := json.Unmarshal(respBody, &responseData); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = f.filesService.AddFile(responseData.UserId, responseData.FileName, responseData.FilePath, responseData.FileType)
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
			http.Error(w, "Incorrect path", http.StatusBadRequest)
			return
		}

		hash := parts[2]

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}
		if !file.ShareStatus {
			respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))
			if err != nil {
				http.Error(w, "Server Internal Error", http.StatusInternalServerError)
				return
			}

			var claims Claims

			if err := json.Unmarshal(respBody, &claims); err != nil {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}

			if claims.UserId != file.User_id {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
		}

		var reqBody struct {
			FilePath string
		}
		reqBody.FilePath = file.FilePath

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		respBody, headers, err := f.ProxyRequest(r, "http://localhost:9995/download", bytes.NewBuffer(jsonData), http.MethodGet)

		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if headers == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Disposition", headers.Get("Content-Disposition"))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", headers.Get("Content-Length"))

		_, err = io.Copy(w, io.Reader(bytes.NewReader(respBody)))
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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
			http.Error(w, "Incorrect path", http.StatusBadRequest)
			return
		}

		hash := parts[2]

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		file, err := f.filesService.GetFileByHash(hash)
		if err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}
		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)

		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		if claims.UserId != file.User_id {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		var reqBody struct {
			FilePath string
		}
		reqBody.FilePath = file.FilePath

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}

		_, _, err = f.ProxyRequest(r, "http://localhost:9995/delete", bytes.NewBuffer(jsonData), http.MethodPost)

		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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
