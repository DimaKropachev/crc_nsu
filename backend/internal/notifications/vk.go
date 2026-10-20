package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Sender interface {
	SendReminder(context.Context, int64, uuid.UUID, string) error
}
type VKMessenger struct {
	token, version string
	client         *http.Client
}

func NewVKMessenger(token, version string) *VKMessenger {
	return &VKMessenger{token: token, version: version, client: &http.Client{Timeout: 8 * time.Second}}
}
func (v *VKMessenger) SendReminder(ctx context.Context, vkID int64, reminderID uuid.UUID, message string) error {
	randomID := int64(crc32.ChecksumIEEE([]byte(reminderID.String())))
	form := url.Values{"user_id": {strconv.FormatInt(vkID, 10)}, "random_id": {strconv.FormatInt(randomID, 10)}, "message": {message}, "access_token": {v.token}, "v": {v.version}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.vk.com/method/messages.send", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var payload struct {
		Response *int64 `json:"response"`
		Error    *struct {
			Code    int    `json:"error_code"`
			Message string `json:"error_msg"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("VK API HTTP %d", resp.StatusCode)
	}
	if payload.Error != nil {
		return fmt.Errorf("VK API error %d", payload.Error.Code)
	}
	if payload.Response == nil {
		return fmt.Errorf("VK API response missing success value")
	}
	return nil
}
