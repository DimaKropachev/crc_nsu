package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Result struct {
	Summary            string   `json:"summary"`
	Score              int      `json:"score"`
	Strengths          []string `json:"strengths"`
	Weaknesses         []string `json:"weaknesses"`
	Recommendations    []string `json:"recommendations"`
	StructureAdvice    []string `json:"structure_advice"`
	ProjectAdvice      []string `json:"project_advice"`
	MissingInformation []string `json:"missing_information"`
}

type Provider interface {
	Analyze(context.Context, string) (Result, error)
}
type OpenAICompatible struct {
	baseURL, apiKey, model string
	client                 *http.Client
}

func NewOpenAICompatible(baseURL, key, model string, timeout time.Duration) *OpenAICompatible {
	return &OpenAICompatible{baseURL: strings.TrimRight(baseURL, "/"), apiKey: key, model: model, client: &http.Client{Timeout: timeout}}
}
func (p *OpenAICompatible) Analyze(ctx context.Context, resumeText string) (Result, error) {
	if strings.TrimSpace(resumeText) == "" {
		return Result{}, fmt.Errorf("resume text is empty")
	}
	if len(resumeText) > 100000 {
		return Result{}, fmt.Errorf("resume text exceeds analysis limit")
	}
	prompt := `Проанализируй резюме для поиска стажировки или junior IT-позиции. Не придумывай факты, которых нет в резюме. Дай конкретные конструктивные советы на русском языке. Верни только JSON-объект по схеме: {"summary":"string","score":0,"strengths":["string"],"weaknesses":["string"],"recommendations":["string"],"structure_advice":["string"],"project_advice":["string"],"missing_information":["string"]}. score — целое число от 0 до 100. Если данных не хватает, отметь это явно.`
	body := map[string]any{"model": p.model, "temperature": 0.2, "response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{{"role": "system", "content": prompt}, {"role": "user", "content": "Текст резюме:\n\n" + resumeText}}}
	encoded, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return Result{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{}, fmt.Errorf("AI provider returned HTTP %d", resp.StatusCode)
	}
	var envelope struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Result{}, err
	}
	if len(envelope.Choices) == 0 {
		return Result{}, fmt.Errorf("AI provider response has no choices")
	}
	var result Result
	if err := json.Unmarshal([]byte(envelope.Choices[0].Message.Content), &result); err != nil {
		return Result{}, fmt.Errorf("invalid AI JSON response: %w", err)
	}
	if result.Score < 0 || result.Score > 100 {
		return Result{}, fmt.Errorf("AI response score is out of range")
	}
	if result.Summary == "" {
		return Result{}, fmt.Errorf("AI response has no summary")
	}
	return result, nil
}
