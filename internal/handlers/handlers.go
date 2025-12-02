package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"

	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder

	"github.com/aswearingen91/Steganography-Service/internal/steg"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

type EncodeResponse struct {
	ImageBase64 string `json:"imageBase64"`
	Filename    string `json:"filename"`
}

type DecodeResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// -------------------------
// Helper: JSON response
// -------------------------
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// -------------------------
// EncodeHandler (multipart/form-data)
// -------------------------
func (h *Handler) EncodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[encode] start")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		log.Println("[encode] could not parse multipart:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid multipart form: " + err.Error()})
		return
	}

	message := r.FormValue("message")
	if message == "" {
		log.Println("[encode] blank message")
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "message cannot be blank"})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Println("[encode] missing image file:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing image file: " + err.Error()})
		return
	}
	defer file.Close()

	log.Printf("[encode] received file: %s (%d bytes)", header.Filename, header.Size)

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("[encode] failed to read image:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not read image: " + err.Error()})
		return
	}

	img, format, err := steg.DecodeImageFromReader(bytes.NewReader(imgBytes))
	if err != nil {
		log.Println("[encode] decode failed:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not decode image: " + err.Error()})
		return
	}

	log.Printf("[encode] image format=%s", format)

	outPNG, err := steg.EmbedBytes(img, []byte(message))
	if err != nil {
		log.Println("[encode] embed failed:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "steganography failed: " + err.Error()})
		return
	}

	resp := EncodeResponse{
		ImageBase64: base64.StdEncoding.EncodeToString(outPNG),
		Filename:    "steg_image.png",
	}

	writeJSON(w, http.StatusOK, resp)
	log.Println("[encode] success")
}

// -------------------------
// DecodeHandler (multipart/form-data) with better error
// -------------------------
func (h *Handler) DecodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[decode] start")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 50<<20)
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		log.Println("[decode] parse multipart failed:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid multipart form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Println("[decode] missing file:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing image file: " + err.Error()})
		return
	}
	defer file.Close()

	log.Printf("[decode] received file: %s (%d bytes)", header.Filename, header.Size)

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		log.Println("[decode] failed to read image:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not read image: " + err.Error()})
		return
	}

	img, format, err := steg.DecodeImageFromReader(bytes.NewReader(imgBytes))
	if err != nil {
		log.Println("[decode] decode failed:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not decode image: " + err.Error()})
		return
	}

	log.Printf("[decode] image format=%s", format)

	// Extract hidden message
	payload, err := steg.ExtractBytes(img)
	if err != nil {
		log.Println("[decode] extract failed:", err)
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "extract failed: " + err.Error(),
		})
		return
	}

	// 🔥 Option 2: Warn user if they used the wrong decode option
	if len(payload) == 0 {
		log.Println("[decode] no hidden message found")
		writeJSON(w, http.StatusBadRequest, ErrorResponse{
			Error: "no hidden message found — make sure you selected the *Steganography* decode option (not normal image decode)",
		})
		return
	}

	writeJSON(w, http.StatusOK, DecodeResponse{Message: string(payload)})
	log.Println("[decode] success")
}
