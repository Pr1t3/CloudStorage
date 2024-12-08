package handler

import (
	"FrontendService/internal/models"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html/template"
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

func ProxyRequest(r *http.Request, target string, reqBody io.Reader, method string) ([]byte, *http.Header, error) {
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
		return nil, nil, errors.New("Status Forbidden")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, &resp.Header, err
}

func LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "./static/login.html")
	})
}

// func AddFileHandler() http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
// 			return
// 		}
// 		http.ServeFile(w, r, "./static/adding_file.html")
// 	})
// }

func RegisterHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		http.ServeFile(w, r, "./static/register.html")
	})
}

func ShowFolderEntities() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))
		bodyResp, _, err := ProxyRequest(r, "http://localhost:9996/"+r.URL.Path, r.Body, http.MethodGet)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		var data struct {
			Files            []models.File
			Folders          []models.Folder
			ParentFolderHash string
		}
		data.ParentFolderHash = ""

		err = json.NewDecoder(io.Reader(bytes.NewReader((bodyResp)))).Decode(&data)
		if err != nil {
			http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
			return
		}

		templates := []string{
			"./static/folders.tmpl",
		}

		ts, err := template.ParseFiles(templates...)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = ts.Execute(w, data)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	})
}

func ShowFile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Ошибка при чтении тела запроса", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		bodyResp, _, err := ProxyRequest(r, "http://localhost:9996"+r.URL.Path, r.Body, http.MethodGet)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		if bodyResp == nil {
			http.Error(w, "404 Page Not Found", http.StatusNotFound)
			return
		}

		var file struct {
			File       models.File
			FolderHash string
		}

		var data struct {
			File       models.File
			FolderHash string
			Format     string
			FileData   string
		}

		err = json.NewDecoder(io.Reader(bytes.NewReader((bodyResp)))).Decode(&file)
		if err != nil {
			http.Error(w, "404 Page Not Found", http.StatusNotFound)
			return
		}
		data.Format = strings.Split(file.File.FileType, "/")[0]
		fileDataBytes, _, err := ProxyRequest(r, "http://localhost:9996/download/"+strings.Split(r.URL.Path, "/")[2], r.Body, http.MethodGet)
		if err != nil {
			log.Println(err)
			http.Error(w, "Error fetching file", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		if data.Format == "image" {
			data.FileData = base64.StdEncoding.EncodeToString(fileDataBytes)
		} else if data.Format == "text" {
			data.FileData = string(fileDataBytes)
		} else {
			data.FileData = ""
		}

		data.File = file.File
		data.FolderHash = file.FolderHash

		templates := []string{
			"./static/show_file.tmpl",
		}

		ts, err := template.ParseFiles(templates...)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = ts.Execute(w, data)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	})
}

func ShowProfile() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
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
		photoData, headers, err := ProxyRequest(r, "http://localhost:9999/get-profile-photo", r.Body, http.MethodGet)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		respBody, _, err := ProxyRequest(r, "http://localhost:9999/get-claims/", r.Body, http.MethodGet)

		if err != nil {
			http.Error(w, "Server Internal Error", http.StatusInternalServerError)
			return
		}

		var claims Claims

		if err := json.Unmarshal(respBody, &claims); err != nil {
			http.Error(w, "Status Forbidden", http.StatusForbidden)
			return
		}

		templates := []string{
			"./static/profile.tmpl",
		}
		ts, err := template.ParseFiles(templates...)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		var data struct {
			Email     string
			PhotoType string
			PhotoData string
		}

		data.PhotoData = base64.StdEncoding.EncodeToString(photoData)
		data.PhotoType = headers.Get("Photo-Type")
		data.Email = claims.Email

		err = ts.Execute(w, data)
		if err != nil {
			log.Println(err)
			http.Error(w, "Failed to execute template", http.StatusInternalServerError)
			return
		}
	})
}
