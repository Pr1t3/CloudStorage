package handler

import (
	"AuthService/internal/models"
	"AuthService/internal/service"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (f *AuthHandler) ProxyRequest(r *http.Request, target string, reqBody io.Reader, method string) ([]byte, *http.Header, error) {
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

type AuthHandler struct {
	authService    *service.AuthService
	forbiddenError error
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService, forbiddenError: errors.New("Status Forbidden")}
}

func (h *AuthHandler) Login() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")

		expirationTime, token, err := h.authService.Login(email, password)
		if err != nil {
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  expirationTime,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
	})
}

func (h *AuthHandler) Register() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		email := r.FormValue("email")
		password := r.FormValue("password")
		hashedPassword, err := service.HashPassword(password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}

		err = h.authService.Register(email, hashedPassword)
		if err != nil {
			http.Error(w, "Failed to register user", http.StatusInternalServerError)
			return
		}

		expirationTime, token, err := h.authService.Login(email, password)
		if err != nil {
			http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    token,
			Expires:  expirationTime,
			HttpOnly: true,
			Secure:   false,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
		})
	})
}

func (h *AuthHandler) Validate() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		token := cookie.Value

		ok, err := models.ValidateToken(token)
		if !ok || err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}

func (h *AuthHandler) GetClaims() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		token := cookie.Value

		claims, err := models.GetClaimsFromToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		} else if claims == nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		res, err := json.Marshal(claims)
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}

		w.Write(res)
	})
}

func (h *AuthHandler) Logout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is illegal", http.StatusMethodNotAllowed)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   "",
			Path:    "/",
			Expires: time.Unix(0, 0),
			MaxAge:  -1,
		})
	})
}

func (h *AuthHandler) GetProfilePhoto() http.Handler {
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
		token, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		claims, err := models.GetClaimsFromToken(token.Value)
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		user, err := h.authService.GetUserByEmail(claims.Email)
		if err != nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			http.Error(w, "Internal Server error", http.StatusInternalServerError)
			return
		}
		var reqBody struct {
			FilePath string
		}

		if !user.Photo_path.Valid {
			reqBody.FilePath = "DefaultPhotoProfile.png"
		} else {
			reqBody.FilePath = user.Photo_path.String
		}

		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			log.Fatal(err)
			return
		}
		respBody, headers, err := h.ProxyRequest(r, "http://localhost:9995/download", bytes.NewBuffer(jsonData), http.MethodGet)

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
		w.Header().Add("Photo-Type", user.Photo_type.String)

		_, err = io.Copy(w, io.Reader(bytes.NewReader(respBody)))
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}

func (h *AuthHandler) ChangePassword() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method is not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		oldPassword := r.FormValue("oldPassword")
		newPassword := r.FormValue("newPassword")
		token, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		claims, err := models.GetClaimsFromToken(token.Value)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		user, err := h.authService.GetUserByEmail(claims.Email)
		if err != nil || user == nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if !service.CheckPassword(user.Password_hash, oldPassword) {
			http.Error(w, "Invalid Credentials", http.StatusForbidden)
			return
		}
		hashedPassword, err := service.HashPassword(newPassword)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		err = h.authService.ChangePassword(claims.Email, hashedPassword)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	})
}

func (f *AuthHandler) UploadPhoto() http.Handler {
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

		token, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		claims, err := models.GetClaimsFromToken(token.Value)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		photoPath := "profile_photos/"
		fileName := strconv.Itoa(claims.UserId)

		r.Header.Add("FilePath", photoPath)
		r.Header.Add("FileName", fileName)
		r.Header.Add("UserId", strconv.Itoa(claims.UserId))

		user, err := f.authService.GetUserByEmail(claims.Email)
		if user.Photo_path.Valid {
			var reqBody struct {
				FilePath string
			}
			reqBody.FilePath = user.Photo_path.String
			jsonData, err := json.Marshal(reqBody)
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			_, _, err = f.ProxyRequest(r, "http://localhost:9995/delete-file", bytes.NewBuffer(jsonData), http.MethodPost)
			r.Body = io.NopCloser(io.Reader(bytes.NewReader(body)))
			if err != nil {
				log.Println(err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		respBody, _, err := f.ProxyRequest(r, "http://localhost:9995/upload/", r.Body, http.MethodPost)

		var responseData struct {
			FileType string
			FileName string
			Size     int64
		}

		if err := json.Unmarshal(respBody, &responseData); err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = f.authService.UploadPhoto(claims.UserId, photoPath+fileName+"."+strings.Split(responseData.FileType, "/")[1], responseData.FileType)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
