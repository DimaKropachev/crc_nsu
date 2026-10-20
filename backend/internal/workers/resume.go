package workers

import (
	"archive/zip"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"

	"career-consultations-backend/internal/ai"
	"career-consultations-backend/internal/repository"
	"career-consultations-backend/internal/storage"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"go.uber.org/zap"
)

type ResumeWorker struct {
	repo     *repository.Postgres
	files    *storage.Local
	provider ai.Provider
	log      *zap.Logger
}

func NewResumeWorker(repo *repository.Postgres, files *storage.Local, provider ai.Provider, log *zap.Logger) *ResumeWorker {
	return &ResumeWorker{repo: repo, files: files, provider: provider, log: log}
}
func (w *ResumeWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processOne(ctx)
		}
	}
}
func (w *ResumeWorker) processOne(ctx context.Context) {
	job, file, ok, err := w.repo.ClaimAnalysis(ctx)
	if err != nil {
		w.log.Error("claim resume analysis failed", zap.Error(err))
		return
	}
	if !ok {
		return
	}
	opened, err := w.files.Open(file.StorageKey)
	if err != nil {
		w.fail(ctx, job.ID, job.AttemptCount, err)
		return
	}
	pathName := opened.Name()
	opened.Close()
	text, err := extractText(pathName, file.MIMEType)
	if err != nil {
		w.fail(ctx, job.ID, job.AttemptCount, err)
		return
	}
	result, err := w.provider.Analyze(ctx, text)
	if err != nil {
		w.fail(ctx, job.ID, job.AttemptCount, err)
		return
	}
	raw, err := json.Marshal(result)
	if err != nil {
		w.fail(ctx, job.ID, job.AttemptCount, err)
		return
	}
	if err := w.repo.CompleteAnalysis(ctx, job.ID, raw); err != nil {
		w.log.Error("save resume analysis failed", zap.String("analysis_id", job.ID.String()), zap.Error(err))
	}
}
func (w *ResumeWorker) fail(ctx context.Context, id uuid.UUID, attempt int, err error) {
	retry := attempt < 3
	if saveErr := w.repo.FailAnalysis(ctx, id, "ANALYSIS_FAILED", retry); saveErr != nil {
		w.log.Error("mark analysis failed", zap.Error(saveErr))
	}
	w.log.Warn("resume analysis failed", zap.String("analysis_id", id.String()), zap.Int("attempt", attempt), zap.Error(err))
}

func extractText(path, mimeType string) (text string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			text = ""
			err = errors.New("document parser rejected the file")
		}
	}()
	switch strings.ToLower(filepath.Ext(path)) {
	case ".docx":
		z, err := zip.OpenReader(path)
		if err != nil {
			return "", err
		}
		defer z.Close()
		for _, f := range z.File {
			if f.UncompressedSize64 > 10<<20 {
				return "", errors.New("DOCX entry exceeds extraction limit")
			}
			if f.Name == "word/document.xml" {
				r, err := f.Open()
				if err != nil {
					return "", err
				}
				defer r.Close()
				decoder := xml.NewDecoder(r)
				var b strings.Builder
				for {
					tok, err := decoder.Token()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						return "", err
					}
					if t, ok := tok.(xml.CharData); ok {
						b.Write([]byte(t))
						b.WriteByte(' ')
						if b.Len() > 100000 {
							return "", errors.New("DOCX extracted text exceeds limit")
						}
					}
				}
				text := strings.TrimSpace(b.String())
				if len(text) == 0 {
					return "", errors.New("DOCX contains no extractable text")
				}
				return text, nil
			}
		}
		return "", errors.New("DOCX document.xml not found")
	case ".pdf":
		f, reader, err := pdf.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		plain, err := reader.GetPlainText()
		if err != nil {
			return "", err
		}
		raw, err := io.ReadAll(io.LimitReader(plain, 1<<20))
		if err != nil {
			return "", err
		}
		text := strings.TrimSpace(string(raw))
		if text == "" {
			return "", errors.New("PDF text extraction failed or file is scanned")
		}
		return text, nil
	default:
		return "", errors.New("unsupported resume extension")
	}
}
