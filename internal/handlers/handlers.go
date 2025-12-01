package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aswearingen91/Steganography-Service/internal/crypto"
	"github.com/aswearingen91/Steganography-Service/internal/steg"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// response helpers
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// EncodeHandler: multipart form: image (file), message (text), password (text)
func (h *Handler) EncodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing image file"})
		return
	}
	defer file.Close()

	password := r.FormValue("password")
	if password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing password"})
		return
	}
	message := r.FormValue("message")
	if message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing message"})
		return
	}

	img, format, err := steg.DecodeImageFromReader(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not decode image: " + err.Error()})
		return
	}
	log.Printf("encode: received image format=%s name=%s", format, header.Filename)

	// encrypt message
	encPayload, err := crypto.EncryptWithPassword([]byte(message), password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed: " + err.Error()})
		return
	}

	// embed encrypted payload
	outPNG, err := steg.EmbedBytes(img, encPayload)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "steganography failed: " + err.Error()})
		return
	}

	// return as downloadable PNG
	outName := strings.TrimSuffix(header.Filename, ".png") + "_steg.png"
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", outName))
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, strings.NewReader(string(outPNG))); err != nil {
		// note: convert []byte to reader without copy — use bytes.NewReader
		// but above is a minimal approach; better:
		// io.Copy(w, bytes.NewReader(outPNG))
		// Fixing below:
		_, _ = w.Write(outPNG)
	}
}

// DecodeHandler: multipart form: image (file), password
func (h *Handler) DecodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad form: " + err.Error()})
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing image file"})
		return
	}
	defer file.Close()

	password := r.FormValue("password")
	if password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing password"})
		return
	}

	img, _, err := steg.DecodeImageFromReader(file)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "could not decode image: " + err.Error()})
		return
	}

	payload, err := steg.ExtractBytes(img)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "extract failed: " + err.Error()})
		return
	}

	plain, err := crypto.DecryptWithPassword(payload, password)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "decrypt failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": string(plain)})
}
