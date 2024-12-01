package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func fetchFile(r *http.Request) ([]byte, error) {
	proxyReq, err := http.NewRequest(http.MethodGet, "http://localhost:9996/download/"+strings.Split(r.URL.Path, "/")[2], r.Body)
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch file: %s", resp.Status)
	}

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return fileData, nil
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

func LoginHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "./static/login.html")
	})
}

func AddFileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "./static/adding_file.html")
	})
}

func RegisterHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		http.ServeFile(w, r, "./static/register.html")
	})
}

func ShowAllFiles() http.Handler {
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
		bodyResp, err := ProxyRequest(r, "http://localhost:9996/files")
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		type File struct {
			ID          int       `json:"ID"`
			Hash        string    `json:"Hash"`
			UserID      int       `json:"User_id"`
			FileName    string    `json:"FileName"`
			FilePath    string    `json:"FilePath"`
			FileType    string    `json:"FileType"`
			ShareStatus bool      `json:"ShareStatus"`
			UploadedAt  time.Time `json:"Uploaded_at"`
		}
		type dataStruct struct {
			Files []File
		}
		var files []File
		err = json.NewDecoder(io.Reader(bytes.NewReader((bodyResp)))).Decode(&files)
		if err != nil {
			http.Error(w, "Failed to decode JSON", http.StatusBadRequest)
			return
		}
		var data = dataStruct{Files: files}

		templates := []string{
			"./static/files.tmpl",
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

		bodyResp, err := ProxyRequest(r, "http://localhost:9996"+r.URL.Path)
		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		if bodyResp == nil {
			http.Error(w, "404 Page Not Found", http.StatusNotFound)
			return
		}

		type File struct {
			ID         int       `json:"ID"`
			UserID     int       `json:"User_id"`
			FileName   string    `json:"FileName"`
			FilePath   string    `json:"FilePath"`
			FileType   string    `json:"FileType"`
			UploadedAt time.Time `json:"Uploaded_at"`
			Format     string
			FileData   string
		}
		type dataStruct struct {
			File File
		}
		var file File
		err = json.NewDecoder(io.Reader(bytes.NewReader((bodyResp)))).Decode(&file)
		if err != nil {
			http.Error(w, "404 Page Not Found", http.StatusNotFound)
			return
		}
		file.Format = strings.Split(file.FileType, "/")[0]

		fileDataBytes, err := fetchFile(r)
		if err != nil {
			http.Error(w, "Error fetching file", http.StatusInternalServerError)
			return
		}

		r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))

		if file.Format == "image" {
			file.FileData = base64.StdEncoding.EncodeToString(fileDataBytes)
		} else if file.Format == "text" {
			file.FileData = string(fileDataBytes)
		} else {
			file.FileData = ""
		}
		var data = dataStruct{File: file}

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
