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
	"strings"
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
		if r.Header.Get("UserId") == "" {
			http.Error(w, "UserId header is required", http.StatusBadRequest)
			return
		}
		if _, err := os.Stat("./uploads/" + r.Header.Get("UserId")); os.IsNotExist(err) {
			err = os.Mkdir("./uploads/"+r.Header.Get("UserId"), os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		filePath := "./uploads/" + r.Header.Get("FilePath")

		err := r.ParseMultipartForm(10 << 20)
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

		if r.Header.Get("FileName") == "" {
			filePath += fileHeader.Filename
		} else {
			filePath += r.Header.Get("FileName")
			filePath += "." + strings.Split(fileHeader.Header.Get("Content-Type"), "/")[1]
		}

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
		info, err := dst.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var res struct {
			FileType string
			FileName string
			Size     int64
		}

		res.FileType = fileHeader.Header.Get("Content-Type")
		res.FileName = fileHeader.Filename
		res.Size = int64(info.Size())
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

		openedFile, err := os.Open("./uploads/" + file.FilePath)
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

		err = os.Remove("./uploads/" + file.FilePath)
		if err != nil {
			log.Println(err, file.FilePath)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *StorageHandler) CreateFolder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		var folder struct {
			FolderPath string
		}
		err = json.Unmarshal(body, &folder)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		folder.FolderPath = "./uploads/" + folder.FolderPath
		if _, err := os.Stat(folder.FolderPath); os.IsNotExist(err) {
			err = os.Mkdir(folder.FolderPath, os.ModePerm)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (h *StorageHandler) DeleteFolder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		var folder struct {
			FolderPath string
		}

		err = json.Unmarshal(body, &folder)

		err = os.RemoveAll("./uploads/" + folder.FolderPath)
		if err != nil {
			log.Println(err, folder.FolderPath)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
