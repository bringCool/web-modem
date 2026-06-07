package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gorilla/mux"
)

func ProxyRegister(r *mux.Router, target string) error {
	targetURL, err := url.Parse(target)
	if err != nil {
		return err
	}
	if targetURL.Scheme == "" || targetURL.Host == "" {
		return fmt.Errorf("invalid API_PROXY_TARGET %q", target)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "proxy request failed: " + err.Error(),
		})
	}

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalHost := req.Host
		originalDirector(req)
		req.Host = targetURL.Host
		req.Header.Set("X-Forwarded-Host", originalHost)
	}

	r.PathPrefix("/api").Handler(proxy)
	r.PathPrefix("/ws").Handler(proxy)
	return nil
}
