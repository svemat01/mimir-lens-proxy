package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/google/uuid"
)

func main() {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:9090" // default value
	}

	debug := os.Getenv("DEBUG") == "true"

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()

		if debug {
			log.Printf("[%s] Processing request: %s %s", requestID, r.Method, r.URL.Path)
		}

		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.URL.Path = "/prometheus" + r.URL.Path

		// Check if the content type is form data
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {

			// Parse the multipart form
			err := r.ParseMultipartForm(32 << 20) // 32MB max memory
			if err != nil {
				if debug {
					log.Printf("[%s] Error parsing form data: %v", requestID, err)
				}
				http.Error(w, "Error parsing form data", http.StatusBadRequest)
				return
			}

			// Convert form data to application/x-www-form-urlencoded
			formValues := url.Values{}
			for key, values := range r.MultipartForm.Value {
				for _, value := range values {
					formValues.Add(key, value)
				}
			}

			// Log form data if debug is enabled
			if debug {
				formDataJSON, _ := json.MarshalIndent(formValues, "", "  ")
				log.Printf("[%s] Form data: %s", requestID, string(formDataJSON))
			}

			// Set the new content type and body
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Body = io.NopCloser(strings.NewReader(formValues.Encode()))
			r.ContentLength = int64(len(formValues.Encode()))

			if debug {
				log.Printf("[%s] Converted multipart form to URL-encoded form", requestID)
			}
		}

		proxy.ServeHTTP(w, r)

		if debug {
			log.Printf("[%s] Request completed", requestID)
		}
	})

	log.Printf("Starting reverse proxy on :8080 to %s (Debug: %v)", targetURL, debug)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
