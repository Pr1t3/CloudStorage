package handler

import (
	"FilesService/internal/models"
	"FilesService/internal/service"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
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
	folderService  *service.FolderService
	forbiddenError error
}

func NewFilesHandler(filesService *service.FilesService, folderService *service.FolderService) *FilesHandler {
	return &FilesHandler{filesService: filesService, folderService: folderService, forbiddenError: errors.New("status forbidden")}
}

func (f *FilesHandler) GetFolderEntities() http.HandlerFunc {
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var data struct {
			Files            []models.File
			Folders          []models.Folder
			ParentFolderHash string
		}
		data.ParentFolderHash = "null"

		path := r.URL.Path

		parts := strings.Split(path, "/")
		var folderHash *string = nil
		var folderId *int = nil
		if len(parts) >= 3 && parts[2] != "" {
			folderHash = &parts[2]
			folder, err := f.folderService.GetFolderByHash(*folderHash)
			if err != nil {
				http.Error(w, "Folder not found", http.StatusNotFound)
				return
			}
			folderId = &folder.Id
			data.ParentFolderHash = ""
			if folder.ParentId.Valid {
				parentFolder, err := f.folderService.GetFolderById(int(folder.ParentId.Int32))
				if err != nil {
					log.Println(err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}
				data.ParentFolderHash = parentFolder.Hash
			}

		}
		files, err := f.filesService.GetFilesInFolder(folderId, claims.UserId)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		folders, err := f.folderService.GetSiblingFolders(folderId, claims.UserId)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		data.Files = files
		data.Folders = folders

		jsonData, err := json.Marshal(data)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
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

		var data struct {
			File       *models.File
			FolderHash string
		}
		data.File = file
		data.FolderHash = ""

		if file.FolderId.Valid {
			folder, err := f.folderService.GetFolderById(int(file.FolderId.Int32))
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			data.FolderHash = folder.Hash
		}

		jsonData, err := json.Marshal(data)
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		hash := r.Header.Get("Hash")
		var folder_id *int = nil
		folderPath := ""
		var filePath string
		if hash != "" {
			folder, err := f.folderService.GetFolderByHash(hash)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			folder_id = &folder.Id
			folderPath = folder.FolderPath
			filePath = folderPath + "/"
		} else {
			filePath = strconv.Itoa(claims.UserId) + "/"
		}

		r.Header.Add("FilePath", filePath)
		r.Header.Add("UserId", strconv.Itoa(claims.UserId))
		respBody, _, err = f.ProxyRequest(r, "http://localhost:9995/upload/", r.Body, http.MethodPost)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var responseData struct {
			FileType string
			FileName string
			Size     int64
		}

		if err := json.Unmarshal(respBody, &responseData); err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = f.filesService.AddFile(claims.UserId, folder_id, responseData.Size, responseData.FileName, responseData.FileType)
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
			respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
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
		if file.FolderId.Valid {
			folder, err := f.folderService.GetFolderById(int(file.FolderId.Int32))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			reqBody.FilePath += folder.FolderPath + "/"
		} else {
			reqBody.FilePath = strconv.Itoa(file.User_id) + "/"
		}
		reqBody.FilePath += file.FileName

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
		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)

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
		reqBody.FilePath = strconv.Itoa(claims.UserId) + "/"
		if file.FolderId.Valid {
			folder, err := f.folderService.GetFolderById(int(file.FolderId.Int32))
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			reqBody.FilePath += folder.FolderPath
		}
		reqBody.FilePath += file.FileName

		reqBody.FilePath = strings.ReplaceAll(reqBody.FilePath, "//", "/")

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}
		_, _, err = f.ProxyRequest(r, "http://localhost:9995/delete-file", bytes.NewBuffer(jsonData), http.MethodPost)

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

func (f *FilesHandler) CreateFolder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var data struct {
			FolderName       string `json:"folderName"`
			ParentFolderHash string `json:"hash"`
		}
		err = json.Unmarshal(body, &data)
		if err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)
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
		var parentId *int = nil
		var folderPath string
		if data.ParentFolderHash != "" {
			folder, err := f.folderService.GetFolderByHash(data.ParentFolderHash)
			if err != nil {
				http.Error(w, "Status Forbidden", http.StatusForbidden)
				return
			}
			parentId = &folder.Id
			folderPath += folder.FolderPath + "/"
		} else {
			folderPath = strconv.Itoa(claims.UserId) + "/"
		}
		folderPath += data.FolderName

		err = f.folderService.CreateFolder(data.FolderName, folderPath, parentId, claims.UserId)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		var reqBody struct {
			FolderPath string
		}
		reqBody.FolderPath = folderPath

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}
		respBody, _, err = f.ProxyRequest(r, "http://localhost:9995/create_folder/", bytes.NewBuffer(jsonData), http.MethodPost)
		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}

func (f *FilesHandler) DeleteFolder() http.Handler {
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

		folder, err := f.folderService.GetFolderByHash(hash)
		if err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}
		respBody, _, err := f.ProxyRequest(r, "http://localhost:9999/get-claims/", nil, http.MethodGet)

		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		if claims.UserId != folder.UserId {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}
		var reqBody struct {
			FolderPath string
		}
		reqBody.FolderPath = folder.FolderPath
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}
		_, _, err = f.ProxyRequest(r, "http://localhost:9995/delete-folder", bytes.NewBuffer(jsonData), http.MethodPost)
		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		err = f.folderService.DeleteFolder(hash)
		if err != nil {
			log.Println(err)
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
