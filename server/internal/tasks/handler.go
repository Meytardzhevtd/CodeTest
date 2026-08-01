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
	r.Get("/slug/{slug}", h.GetTaskBySlug)
	r.Get("/{id}", h.GetTaskByID)
	r.Put("/{id}", h.UpdateTask)
	r.Delete("/{id}", h.DeleteTask)
	r.Post("/{id}/tests", h.UploadTests)
	r.Post("/{id}/tags", h.AddTags)
	r.Put("/{id}/tags", h.SetTags)
	r.Post("/{id}/examples", h.SetExamples)

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

func (h *Handler) GetTaskBySlug(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}

	task, err := h.service.GetTaskBySlug(r.Context(), slug)
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

func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
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

	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	task, err := h.service.UpdateTask(r.Context(), userId, taskID, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "only the task creator can edit the task")
		default:
			writeError(w, http.StatusBadRequest, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, task)
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

func (h *Handler) AddTags(w http.ResponseWriter, r *http.Request) {
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

	var req AddTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tags, err := h.service.AddTagsToTask(r.Context(), userId, taskID, req.Tags)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "only the task creator can add tags")
		case errors.Is(err, ErrNoTagsProvided), errors.Is(err, ErrInvalidTagName):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, AddTagsResponse{Tags: tags})
}

func (h *Handler) SetTags(w http.ResponseWriter, r *http.Request) {
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

	var req AddTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tags, err := h.service.SetTags(r.Context(), userId, taskID, req.Tags)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "only the task creator can edit tags")
		case errors.Is(err, ErrInvalidTagName):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, AddTagsResponse{Tags: tags})
}

func (h *Handler) SetExamples(w http.ResponseWriter, r *http.Request) {
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

	var req SetExamplesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	examples, err := h.service.SetExamples(r.Context(), userId, taskID, req.Examples)
	if err != nil {
		switch {
		case errors.Is(err, ErrTaskNotFound):
			writeError(w, http.StatusNotFound, "task not found")
		case errors.Is(err, ErrForbidden):
			writeError(w, http.StatusForbidden, "only the task creator can set examples")
		case errors.Is(err, ErrTooManyExamples):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}

	writeJSON(w, http.StatusOK, SetExamplesResponse{Examples: examples})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
