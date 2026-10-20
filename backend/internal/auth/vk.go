package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidVKSignature = errors.New("invalid VK launch signature")

// VerifyLaunchParams validates VK Mini Apps launch params using the app secret.
// Only vk_* parameters are signed. The sign field is not included in the digest.
func VerifyLaunchParams(raw, appID, secret string, now time.Time) (url.Values, error) {
	raw = strings.TrimPrefix(raw, "?")
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, ErrInvalidVKSignature
	}
	for key, vals := range values {
		if len(vals) != 1 {
			return nil, ErrInvalidVKSignature
		}
		_ = key
	}
	sign := values.Get("sign")
	userID := values.Get("vk_user_id")
	if sign == "" || userID == "" || values.Get("vk_app_id") == "" {
		return nil, ErrInvalidVKSignature
	}
	if appID != "" && values.Get("vk_app_id") != appID {
		return nil, ErrInvalidVKSignature
	}
	if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
		return nil, ErrInvalidVKSignature
	}

	keys := make([]string, 0, len(values))
	for key := range values {
		if strings.HasPrefix(key, "vk_") {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, ErrInvalidVKSignature
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(url.QueryEscape(key))
		canonical.WriteByte('=')
		canonical.WriteString(url.QueryEscape(values.Get(key)))
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical.String()))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sign), []byte(expected)) {
		return nil, ErrInvalidVKSignature
	}

	// Current clients may provide vk_ts. When present, reject stale/future launch data.
	if ts := values.Get("vk_ts"); ts != "" {
		seconds, parseErr := strconv.ParseInt(ts, 10, 64)
		if parseErr != nil {
			return nil, ErrInvalidVKSignature
		}
		age := now.Sub(time.Unix(seconds, 0))
		if age > 24*time.Hour || age < -5*time.Minute {
			return nil, ErrInvalidVKSignature
		}
	}
	return values, nil
}
