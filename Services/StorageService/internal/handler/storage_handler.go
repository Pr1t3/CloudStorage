package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type StorageHandler struct {
	forbiddenError error
}

func NewStorageHandler() *StorageHandler {
	return &StorageHandler{forbiddenError: errors.New("Status Forbidden")}
}

type Claims struct {
	Email  string
	UserId int
	exp    time.Time
	iat    time.Time
}

func (h *StorageHandler) ProxyRequest(r *http.Request, target string, method string) ([]byte, error) {
	proxyReq, err := http.NewRequest(method, target, r.Body)
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
	if resp.StatusCode != http.StatusOK {
		return nil, h.forbiddenError
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))
	return body, nil
}

func (h *StorageHandler) UploadFile() http.Handler {
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

		respBody, err := h.ProxyRequest(r, "http://localhost:9999/get-claims/", http.MethodGet)
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
		if _, err := os.Stat("./uploads/"); os.IsNotExist(err) {
			err = os.Mkdir("./uploads/", os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
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
		var res struct {
			UserId   int
			FileName string
			FilePath string
			FileType string
		}
		res.UserId = userId
		res.FileName = fileName
		res.FilePath = filePath
		res.FileType = fileType
		result, err := json.Marshal(res)
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(result)))
		w.Write(result)
	})
}

func (h *StorageHandler) UploadFileOnExactPlace() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}

		if _, err := os.Stat("./uploads/"); os.IsNotExist(err) {
			err = os.Mkdir("./uploads/", os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		if _, err := os.Stat("./uploads/profile_photos/"); os.IsNotExist(err) {
			err = os.Mkdir("./uploads/profile_photos/", os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		filePath := r.Header.Get("FilePath")
		dst, err := os.Create(filePath)
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при сохранении файла", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

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

		_, err = io.Copy(dst, file)
		if err != nil {
			log.Println(err)
			http.Error(w, "Ошибка при копировании файла", http.StatusInternalServerError)
			return
		}

		var res struct {
			FileType string
		}

		res.FileType = fileHeader.Header.Get("Content-Type")
		result, err := json.Marshal(res)
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(result)))
		w.Write(result)
	})
}

func (h *StorageHandler) DownloadFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		var file struct {
			FilePath string
		}

		err = json.Unmarshal(body, &file)

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

func (h *StorageHandler) DeleteFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		var file struct {
			FilePath string
		}

		err = json.Unmarshal(body, &file)

		err = os.Remove(file.FilePath)
		if err != nil {
			log.Println(err, file.FilePath)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
