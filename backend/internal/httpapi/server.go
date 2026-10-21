package httpapi

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"career-consultations-backend/internal/auth"
	"career-consultations-backend/internal/config"
	"career-consultations-backend/internal/models"
	"career-consultations-backend/internal/repository"
	"career-consultations-backend/internal/service"
	"career-consultations-backend/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Server struct {
	cfg           config.Config
	repo          *repository.Postgres
	consultations *service.Consultations
	tokens        *auth.TokenService
	files         *storage.Local
	logger        *zap.Logger
	aiReady       bool
	limiter       *ipLimiter
}
type principal struct{ User models.User }
type contextKey string

const principalKey contextKey = "principal"

func New(cfg config.Config, repo *repository.Postgres, tokens *auth.TokenService, files *storage.Local, logger *zap.Logger, aiReady bool) *Server {
	return &Server{cfg: cfg, repo: repo, consultations: service.NewConsultations(repo), tokens: tokens, files: files, logger: logger, aiReady: aiReady, limiter: newIPLimiter(30, time.Minute)}
}
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(s.requestLogger)
	r.Use(recoverer(s.logger))
	r.Use(s.cors)
	r.Get("/api/v1/health/live", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/api/v1/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := s.repo.Pool.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "NOT_READY", "База данных недоступна", nil)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	r.With(s.limiter.Middleware).Post("/api/v1/auth/vk", s.authVK)
	r.Group(func(r chi.Router) {
		r.Use(s.authenticate)
		r.Get("/api/v1/me", s.me)
		r.Post("/api/v1/me/notification-consent", s.setNotificationConsent)
		r.Get("/api/v1/consultants", s.listConsultants)
		r.Get("/api/v1/consultants/{id}", s.getConsultant)
		r.Get("/api/v1/consultants/{id}/slots", s.listConsultantSlots)
		r.Post("/api/v1/consultations", s.createConsultation)
		r.Get("/api/v1/consultations/my", s.listMyConsultations)
		r.Get("/api/v1/consultations/{id}", s.getConsultation)
		r.Post("/api/v1/consultations/{id}/reschedule", s.rescheduleConsultation)
		r.Post("/api/v1/consultations/{id}/cancel", s.cancelConsultation)
		r.With(s.limiter.Middleware).Post("/api/v1/files/resumes", s.uploadResume)
		r.Get("/api/v1/files/{id}", s.getFile)
		r.Delete("/api/v1/files/{id}", s.deleteFile)
		r.Post("/api/v1/resume-analyses", s.createAnalysis)
		r.Get("/api/v1/resume-analyses", s.listAnalyses)
		r.Get("/api/v1/resume-analyses/{id}", s.getAnalysis)
		r.Group(func(r chi.Router) {
			r.Use(s.requireRole("admin"))
			r.Get("/api/v1/admin/consultations", s.listAdminConsultations)
			r.Post("/api/v1/admin/consultations/{id}/confirm", s.confirmConsultation)
			r.Post("/api/v1/admin/consultations/{id}/complete", s.completeConsultation)
			r.Get("/api/v1/admin/slots", s.listAdminSlots)
			r.Post("/api/v1/admin/slots", s.createSlot)
			r.Patch("/api/v1/admin/slots/{id}", s.updateSlot)
			r.Post("/api/v1/admin/slots/{id}/block", s.blockSlot)
			r.Get("/api/v1/admin/profile", s.getAdminProfile)
			r.Patch("/api/v1/admin/profile", s.updateAdminProfile)
		})
	})
	return r
}
func (s *Server) authVK(w http.ResponseWriter, r *http.Request) {
	if s.cfg.VKAppID == "" || s.cfg.VKAppSecret == "" {
		writeError(w, http.StatusServiceUnavailable, "VK_AUTH_NOT_CONFIGURED", "Авторизация через VK не настроена на сервере", nil)
		return
	}
	var req struct {
		LaunchParams string `json:"launch_params"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректное тело запроса", nil)
		return
	}
	if len(req.LaunchParams) == 0 || len(req.LaunchParams) > 16*1024 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "launch_params отсутствует или слишком длинный", nil)
		return
	}
	values, err := auth.VerifyLaunchParams(req.LaunchParams, s.cfg.VKAppID, s.cfg.VKAppSecret, time.Now())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "INVALID_VK_SIGNATURE", "Параметры запуска VK не прошли проверку подписи", nil)
		return
	}
	vkID, err := strconv.ParseInt(values.Get("vk_user_id"), 10, 64)
	if err != nil || vkID <= 0 {
		writeError(w, http.StatusUnauthorized, "INVALID_VK_SIGNATURE", "Некорректный VK ID", nil)
		return
	}
	_, admin := s.cfg.AdminVKIDs[vkID]
	user, err := s.repo.UpsertVKUser(r.Context(), vkID, values.Get("vk_first_name"), values.Get("vk_last_name"), firstNonEmpty(values.Get("vk_photo_200"), values.Get("vk_photo_100")), admin)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	token, expires, err := s.tokens.Issue(user.ID, user.VKID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": expires, "user": user})
}
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Требуется Bearer access token", nil)
			return
		}
		id, vkID, err := s.tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Токен недействителен или истёк", nil)
			return
		}
		user, err := s.repo.GetUser(r.Context(), id)
		if err != nil || user.VKID != vkID {
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Пользователь токена не найден", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, principal{User: user})))
	})
}
func (s *Server) requireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := getPrincipal(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Необходима авторизация", nil)
				return
			}
			if p.User.Role != role {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "Недостаточно прав для выполнения операции", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	writeJSON(w, http.StatusOK, p.User)
}
func (s *Server) setNotificationConsent(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Ожидается поле enabled типа boolean", nil)
		return
	}
	user, err := s.repo.SetNotificationConsent(r.Context(), p.User.ID, req.Enabled)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"notification_consent": user.NotificationConsent, "notification_consent_at": user.NotificationConsentAt})
}
func (s *Server) listConsultants(w http.ResponseWriter, r *http.Request) {
	items, err := s.repo.ListConsultants(r.Context())
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) getConsultant(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	item, err := s.repo.GetConsultant(r.Context(), id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) listConsultantSlots(w http.ResponseWriter, r *http.Request) {
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	from, to, ok := dateRange(w, r, 90*24*time.Hour)
	if !ok {
		return
	}
	items, err := s.repo.ListSlots(r.Context(), id, from, to)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) createConsultation(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	if p.User.Role != "student" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Записаться на консультацию могут только студенты", nil)
		return
	}
	var req struct {
		SlotID             uuid.UUID  `json:"slot_id"`
		ProblemDescription string     `json:"problem_description"`
		ResumeFileID       *uuid.UUID `json:"resume_file_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.SlotID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Нужно указать корректный slot_id", nil)
		return
	}
	req.ProblemDescription = strings.TrimSpace(req.ProblemDescription)
	if len(req.ProblemDescription) > 5000 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Описание проблемы не должно превышать 5000 символов", nil)
		return
	}
	c, err := s.consultations.Create(r.Context(), p.User.ID, req.SlotID, req.ProblemDescription, req.ResumeFileID)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}
func (s *Server) listMyConsultations(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	items, err := s.repo.ListMyConsultations(r.Context(), p.User.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) getConsultation(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	c, err := s.repo.GetConsultation(r.Context(), p.User.ID, p.User.Role, id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s *Server) rescheduleConsultation(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	if p.User.Role != "student" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Переносить консультации могут только студенты", nil)
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		NewSlotID uuid.UUID `json:"new_slot_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.NewSlotID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Нужно указать корректный new_slot_id", nil)
		return
	}
	c, err := s.consultations.Reschedule(r.Context(), p.User.ID, id, req.NewSlotID)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s *Server) cancelConsultation(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	if p.User.Role != "student" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Отменять записи могут только студенты", nil)
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.consultations.Cancel(r.Context(), p.User.ID, id); err != nil {
		s.repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) listAdminConsultations(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	from, to, ok := dateRange(w, r, 180*24*time.Hour)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && !oneOf(status, "pending", "confirmed", "completed", "cancelled") {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Неизвестный статус консультации", nil)
		return
	}
	limit := intQuery(r, "limit", 50)
	offset := intQuery(r, "offset", 0)
	if limit < 1 || limit > 100 || offset < 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "limit должен быть 1..100, offset >= 0", nil)
		return
	}
	items, err := s.repo.ListAdminConsultations(r.Context(), p.User.ID, status, from, to, limit, offset)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}
func (s *Server) confirmConsultation(w http.ResponseWriter, r *http.Request) {
	s.changeConsultationStatus(w, r, "confirmed")
}
func (s *Server) completeConsultation(w http.ResponseWriter, r *http.Request) {
	s.changeConsultationStatus(w, r, "completed")
}
func (s *Server) changeConsultationStatus(w http.ResponseWriter, r *http.Request, target string) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.consultations.ChangeStatus(r.Context(), p.User.ID, id, target); err != nil {
		s.repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) listAdminSlots(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	from, to, ok := dateRange(w, r, 180*24*time.Hour)
	if !ok {
		return
	}
	items, err := s.repo.ListAdminSlots(r.Context(), p.User.ID, from, to)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) createSlot(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	var req struct {
		StartsAt time.Time `json:"starts_at"`
		EndsAt   time.Time `json:"ends_at"`
	}
	if err := decodeJSON(r, &req); err != nil || !req.EndsAt.After(req.StartsAt) || !req.StartsAt.After(time.Now()) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Укажите корректные даты будущего слота в формате RFC3339", nil)
		return
	}
	if req.EndsAt.Sub(req.StartsAt) > 4*time.Hour {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Продолжительность слота не может превышать 4 часа", nil)
		return
	}
	item, err := s.consultations.CreateSlot(r.Context(), p.User.ID, req.StartsAt, req.EndsAt)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) updateSlot(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		StartsAt time.Time `json:"starts_at"`
		EndsAt   time.Time `json:"ends_at"`
	}
	if err := decodeJSON(r, &req); err != nil || !req.EndsAt.After(req.StartsAt) || !req.StartsAt.After(time.Now()) {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Укажите корректные даты будущего слота в формате RFC3339", nil)
		return
	}
	item, err := s.consultations.UpdateSlot(r.Context(), p.User.ID, id, req.StartsAt, req.EndsAt)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (s *Server) blockSlot(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	if err := s.consultations.BlockSlot(r.Context(), p.User.ID, id); err != nil {
		s.repoError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) getAdminProfile(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	c, err := s.repo.GetAdminProfile(r.Context(), p.User.ID)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}
func (s *Server) updateAdminProfile(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	var req struct {
		Specialization string `json:"specialization"`
		Description    string `json:"description"`
	}
	if err := decodeJSON(r, &req); err != nil || len(req.Specialization) > 200 || len(req.Description) > 3000 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный профиль консультанта", nil)
		return
	}
	c, err := s.repo.UpdateAdminProfile(r.Context(), p.User.ID, strings.TrimSpace(req.Specialization), strings.TrimSpace(req.Description))
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) uploadResume(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	if p.User.Role != "student" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Загружать резюме могут только студенты", nil)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+1024*1024)
	if err := r.ParseMultipartForm(s.cfg.MaxUploadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Файл превышает допустимый размер", nil)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()
	f, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "В multipart-запросе требуется поле file", nil)
		return
	}
	defer f.Close()
	if header.Size <= 0 || header.Size > s.cfg.MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Файл пуст или превышает допустимый размер", nil)
		return
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".pdf" && ext != ".docx" {
		writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Поддерживаются только PDF и DOCX", nil)
		return
	}
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	head = head[:n]
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", "Не удалось прочитать файл", nil)
		return
	}
	if ext == ".docx" {
		zr, zipErr := zip.NewReader(f, header.Size)
		if zipErr != nil {
			writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Файл не является корректным DOCX", nil)
			return
		}
		hasDocumentXML := false
		for _, entry := range zr.File {
			if entry.Name == "word/document.xml" {
				hasDocumentXML = true
			}
			if entry.UncompressedSize64 > 10<<20 {
				writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Внутренний компонент DOCX слишком велик", nil)
				return
			}
		}
		if !hasDocumentXML {
			writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "В DOCX отсутствует документ", nil)
			return
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			writeError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", "Не удалось прочитать файл", nil)
			return
		}
	}
	mimeType := ""
	if ext == ".pdf" && strings.HasPrefix(string(head), "%PDF-") {
		mimeType = "application/pdf"
	} else if ext == ".docx" && len(head) >= 4 && head[0] == 'P' && head[1] == 'K' {
		mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	} else {
		writeError(w, http.StatusUnsupportedMediaType, "INVALID_FILE_TYPE", "Содержимое файла не соответствует расширению", nil)
		return
	}
	key, size, err := s.files.Save(f, ext, s.cfg.MaxUploadBytes)
	if err != nil {
		if strings.Contains(err.Error(), "too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "Файл превышает допустимый размер", nil)
		} else {
			s.internalError(w, r, err)
		}
		return
	}
	id := uuid.New()
	name := filepath.Base(header.Filename)
	if len(name) > 255 {
		name = name[len(name)-255:]
	}
	item, err := s.repo.CreateFile(r.Context(), models.UploadedFile{ID: id, OwnerUserID: p.User.ID, OriginalFilename: name, StorageKey: key, MIMEType: mimeType, SizeBytes: size})
	if err != nil {
		_ = s.files.Delete(key)
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (s *Server) getFile(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	f, err := s.repo.GetFileAccessible(r.Context(), id, p.User.ID, p.User.Role)
	if err != nil {
		s.repoError(w, err)
		return
	}
	file, err := s.files.Open(f.StorageKey)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", f.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_ = mime.FormatMediaType("attachment", map[string]string{"filename": f.OriginalFilename})
	w.Header().Set("Content-Length", strconv.FormatInt(f.SizeBytes, 10))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}
func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	f, err := s.repo.DeleteFile(r.Context(), id, p.User.ID)
	if err != nil {
		s.repoError(w, err)
		return
	}
	if err := s.files.Delete(f.StorageKey); err != nil {
		s.logger.Warn("delete stored file failed", zap.String("file_id", id.String()), zap.Error(err))
	}
	w.WriteHeader(http.StatusNoContent)
}
func (s *Server) createAnalysis(w http.ResponseWriter, r *http.Request) {
	if !s.aiReady {
		writeError(w, http.StatusServiceUnavailable, "ANALYSIS_PROVIDER_UNAVAILABLE", "AI-анализ сейчас не настроен на сервере", nil)
		return
	}
	p, _ := getPrincipal(r.Context())
	var req struct {
		FileID            uuid.UUID `json:"file_id"`
		ExternalAIConsent bool      `json:"external_ai_consent"`
	}
	if err := decodeJSON(r, &req); err != nil || req.FileID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Нужно указать file_id", nil)
		return
	}
	if !req.ExternalAIConsent {
		writeError(w, http.StatusUnprocessableEntity, "EXTERNAL_AI_CONSENT_REQUIRED", "Для запуска анализа необходимо согласие на обработку текста резюме внешним ИИ-провайдером", nil)
		return
	}
	a, err := s.repo.CreateAnalysis(r.Context(), p.User.ID, req.FileID, req.ExternalAIConsent)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, a)
}
func (s *Server) listAnalyses(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	items, err := s.repo.ListAnalyses(r.Context(), p.User.ID)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) getAnalysis(w http.ResponseWriter, r *http.Request) {
	p, _ := getPrincipal(r.Context())
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	a, err := s.repo.GetAnalysis(r.Context(), p.User.ID, id)
	if err != nil {
		s.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) repoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Ресурс не найден", nil)
	case errors.Is(err, repository.ErrForbidden):
		writeError(w, http.StatusForbidden, "FORBIDDEN", "Нет доступа к ресурсу", nil)
	case errors.Is(err, repository.ErrSlotUnavailable):
		writeError(w, http.StatusConflict, "SLOT_UNAVAILABLE", "Выбранное время уже недоступно", nil)
	case errors.Is(err, repository.ErrRescheduleLimit):
		writeError(w, http.StatusConflict, "RESCHEDULE_LIMIT_REACHED", "Лимит переносов исчерпан", nil)
	case errors.Is(err, repository.ErrConsentRequired):
		writeError(w, http.StatusUnprocessableEntity, "EXTERNAL_AI_CONSENT_REQUIRED", "Необходимо явное согласие на обработку резюме", nil)
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Данные не соответствуют бизнес-правилам", nil)
	case errors.Is(err, repository.ErrConflict):
		writeError(w, http.StatusConflict, "CONFLICT", "Операция конфликтует с текущим состоянием ресурса", nil)
	default:
		s.logger.Error("request failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
	}
}
func (s *Server) internalError(w http.ResponseWriter, r *http.Request, err error) {
	s.logger.Error("internal error", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Error(err))
	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
}
func (s *Server) requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		s.logger.Info("http request", zap.String("method", r.Method), zap.String("path", r.URL.Path), zap.Int("status", ww.status), zap.Duration("duration", time.Since(started)))
	})
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := false
			for _, candidate := range s.cfg.AllowedOrigins {
				if candidate == origin {
					allowed = true
					break
				}
			}
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			} else if s.cfg.Env == "development" && isLocalOrigin(origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			} else {
				writeError(w, http.StatusForbidden, "CORS_ORIGIN_DENIED", "Origin не разрешён", nil)
				return
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func recoverer(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic recovered", zap.Any("panic", v))
					writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка сервера", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func decodeJSON(r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return fmt.Errorf("request must contain one JSON object")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	if details == nil {
		details = map[string]any{}
	}
	writeJSON(w, status, models.APIError{Error: models.APIErrorDetail{Code: code, Message: message, Details: details}})
}
func pathUUID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный идентификатор ресурса", nil)
		return uuid.Nil, false
	}
	return id, true
}
func getPrincipal(ctx context.Context) (principal, bool) {
	p, ok := ctx.Value(principalKey).(principal)
	return p, ok
}
func dateRange(w http.ResponseWriter, r *http.Request, max time.Duration) (time.Time, time.Time, bool) {
	now := time.Now().UTC()
	from := now
	to := now.Add(30 * 24 * time.Hour)
	var err error
	if raw := r.URL.Query().Get("from"); raw != "" {
		from, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "from должен быть в формате RFC3339", nil)
			return time.Time{}, time.Time{}, false
		}
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		to, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "to должен быть в формате RFC3339", nil)
			return time.Time{}, time.Time{}, false
		}
	}
	if !to.After(from) || to.Sub(from) > max {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный диапазон дат", nil)
		return time.Time{}, time.Time{}, false
	}
	return from.UTC(), to.UTC(), true
}
func intQuery(r *http.Request, key string, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil {
		return def
	}
	return n
}
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
func oneOf(value string, choices ...string) bool {
	for _, c := range choices {
		if value == c {
			return true
		}
	}
	return false
}
func isLocalOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

type ipEntry struct {
	start time.Time
	count int
}
type ipLimiter struct {
	mu     sync.Mutex
	items  map[string]ipEntry
	limit  int
	window time.Duration
}

func newIPLimiter(limit int, window time.Duration) *ipLimiter {
	return &ipLimiter{items: make(map[string]ipEntry), limit: limit, window: window}
}
func (l *ipLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		now := time.Now()
		l.mu.Lock()
		item := l.items[host]
		if now.Sub(item.start) >= l.window {
			item = ipEntry{start: now}
		}
		item.count++
		l.items[host] = item
		over := item.count > l.limit
		if len(l.items) > 10000 {
			for key, value := range l.items {
				if now.Sub(value.start) > 2*l.window {
					delete(l.items, key)
				}
			}
		}
		l.mu.Unlock()
		if over {
			writeError(w, http.StatusTooManyRequests, "RATE_LIMITED", "Слишком много запросов", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
