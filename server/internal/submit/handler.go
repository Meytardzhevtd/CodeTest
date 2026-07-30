package submit

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/meytardzhevtd/CodeTest/server/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/", h.Submit)
	r.Get("/", h.GetInfoAboutSubmit)
	r.Get("/history", h.GetSubmissionHistory)

	return r
}

func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req := CreateSubmissionRequest{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	repsonse, err := h.service.CreateSubmition(r.Context(), userId, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, repsonse)
}

func (h *Handler) GetInfoAboutSubmit(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	submitID := r.URL.Query().Get("id")
	if submitID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	response, err := h.service.GetInfoAboutSubmit(r.Context(), userId, submitID)
	if err != nil {
		switch {
		case errors.Is(err, ErrSubmissionNotFound):
			writeError(w, http.StatusNotFound, "submission not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "forbidden")
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) GetSubmissionHistory(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	response, err := h.service.GetSubmissionHistory(r.Context(), userId, taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
