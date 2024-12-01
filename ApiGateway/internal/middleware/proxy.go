package middleware

import (
	"io"
	"net/http"
)

func ProxyMiddleware(next http.Handler, target string) http.Handler {
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

		client := &http.Client{}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "Failed to connect to service", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)

		next.ServeHTTP(w, r)
	})
}
