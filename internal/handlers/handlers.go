package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/aswearingen91/Steganography-Service/internal/steg"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// -------------------------
// Request/Response structs
// -------------------------

type EncodeRequest struct {
	Image   string `json:"image"`   // base64 (NO prefix)
	Message string `json:"message"` // text to embed
}

type EncodeResponse struct {
	ImageBase64 string `json:"imageBase64"`
	Filename    string `json:"filename"`
}

type DecodeRequest struct {
	Image string `json:"image"` // base64 (NO prefix)
}

type DecodeResponse struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// helper: JSON response
func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

// -------------------------
// EncodeHandler
// -------------------------
func (h *Handler) EncodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[encode] start")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	var req EncodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	if strings.TrimSpace(req.Message) == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "message cannot be blank"})
		return
	}

	imgData, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64: " + err.Error()})
		return
	}

	img, format, err := steg.DecodeImageFromReader(bytes.NewReader(imgData))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not read image: " + err.Error()})
		return
	}

	log.Printf("[encode] image format=%s", format)

	outPNG, err := steg.EmbedBytes(img, []byte(req.Message))
	if err != nil {
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
// DecodeHandler
// -------------------------
func (h *Handler) DecodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[decode] start")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "use POST"})
		return
	}

	var req DecodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid JSON: " + err.Error()})
		return
	}

	imgData, err := base64.StdEncoding.DecodeString(req.Image)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid base64: " + err.Error()})
		return
	}

	img, format, err := steg.DecodeImageFromReader(bytes.NewReader(imgData))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "could not decode image: " + err.Error()})
		return
	}

	log.Printf("[decode] image format=%s", format)

	payload, err := steg.ExtractBytes(img)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "extract failed: " + err.Error()})
		return
	}

	message := string(payload)
	writeJSON(w, http.StatusOK, DecodeResponse{Message: message})
	log.Println("[decode] success")
}
