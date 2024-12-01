package handler

import (
	"io"
	"net/http"
)

func ProxyHandlerRedirect(target, targetRedir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyReq, err := http.NewRequest(r.Method, target+r.URL.Path, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to connect to service", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}

		io.Copy(w, resp.Body)
		http.Redirect(w, r, targetRedir, http.StatusSeeOther)
	})
}

func ProxyHandler(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyReq, err := http.NewRequest(r.Method, target+r.URL.Path, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
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
			http.Error(w, "Failed to connect to service", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Set(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
