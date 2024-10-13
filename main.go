package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		targetURL = "http://localhost:9090" // default value
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatal(err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/prometheus" + r.URL.Path

		// Check if the content type is form data
		contentType := r.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "multipart/form-data") {
			// Parse the multipart form
			err := r.ParseMultipartForm(32 << 20) // 32MB max memory
			if err != nil {
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

			// Set the new content type and body
			r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			r.Body = io.NopCloser(strings.NewReader(formValues.Encode()))
			r.ContentLength = int64(len(formValues.Encode()))
		}

		proxy.ServeHTTP(w, r)
	})

	log.Println("Starting reverse proxy on :8080 to " + targetURL)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
