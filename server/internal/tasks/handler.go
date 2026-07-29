package tasks

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/meytardzhevtd/CodeTest/server/internal/auth"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/", h.CreateTask)
	r.Get("/", h.ListTasks)
	r.Get("/{id}", h.GetTaskByID)
	r.Delete("/{id}", h.DeleteTask)
	r.Post("/{id}/tests", h.UploadTests)

	return r
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.service.CreateTask(r.Context(), userId, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSlugAlreadyExists):
			writeError(w, http.StatusConflict, "slug already exists")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	task, err := h.service.GetTaskByID(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 10
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	tasks, total, err := h.service.ListTasks(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "something went wrong")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":  tasks,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	if err := h.service.DeleteTask(r.Context(), id); err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusNoContent, nil)
}

func (h *Handler) UploadTests(w http.ResponseWriter, r *http.Request) {
	userId, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	taskID := chi.URLParam(r, "id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "id is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxTestsArchiveBytes)
	if err := r.ParseMultipartForm(MaxTestsArchiveBytes); err != nil {
		writeError(w, http.StatusBadRequest, "archive is too large or the request is not a valid multipart form")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, header, err := r.FormFile("archive")
	if err != nil {
		writeError(w, http.StatusBadRequest, `missing "archive" file field`)
		return
	}
	defer file.Close()

	count, err := h.service.UploadTests(r.Context(), userId, taskID, file, header.Size)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "only the task creator can upload tests")
		case errors.Is(err, ErrInvalidArchive):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tests_uploaded": count})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
