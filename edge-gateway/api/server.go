package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"

	"edge-gateway/device"
	"edge-gateway/gateway"
)

type Server struct {
	gateway *gateway.Gateway
}

func NewServer(gateway *gateway.Gateway) *Server {
	return &Server{gateway: gateway}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/devices", s.handleDevices)
	mux.HandleFunc("/voice", s.handleVoice)
	return mux
}

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listDevices(w, r.Context())
	case http.MethodPost:
		s.registerDevice(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) listDevices(w http.ResponseWriter, ctx context.Context) {
	devices, err := s.gateway.ListDevices(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"devices": devices})
}

func (s *Server) registerDevice(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var dev device.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeError(w, http.StatusBadRequest, "invalid device payload")
		return
	}
	if err := s.gateway.RegisterDevice(r.Context(), dev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, dev)
}

func (s *Server) handleVoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var payload struct {
		MIMEType    string `json:"mime_type"`
		AudioBase64 string `json:"audio_base64"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid voice payload")
		return
	}
	if payload.AudioBase64 == "" {
		writeError(w, http.StatusBadRequest, "audio_base64 is required")
		return
	}

	audio, err := base64.StdEncoding.DecodeString(payload.AudioBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "audio_base64 is invalid")
		return
	}

	results, err := s.gateway.HandleVoice(r.Context(), audio, payload.MIMEType)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, context.Canceled) {
			status = http.StatusRequestTimeout
		}
		writeError(w, status, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
