package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"
)

func signTestValues(v url.Values, secret string) string {
	keys := make([]string, 0, len(v))
	for key := range v {
		if strings.HasPrefix(key, "vk_") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var canonical strings.Builder
	for i, key := range keys {
		if i > 0 {
			canonical.WriteByte('&')
		}
		canonical.WriteString(url.QueryEscape(key))
		canonical.WriteByte('=')
		canonical.WriteString(url.QueryEscape(v.Get(key)))
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(canonical.String()))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyLaunchParamsValid(t *testing.T) {
	values := url.Values{"vk_app_id": {"123"}, "vk_user_id": {"456"}, "vk_first_name": {"Dima Popov"}, "vk_platform": {"android"}}
	values.Set("sign", signTestValues(values, "test-secret"))
	parsed, err := VerifyLaunchParams(values.Encode(), "123", "test-secret", time.Now())
	if err != nil {
		t.Fatalf("valid launch params rejected: %v", err)
	}
	if parsed.Get("vk_user_id") != "456" {
		t.Fatalf("unexpected VK ID: %q", parsed.Get("vk_user_id"))
	}
}

func TestVerifyLaunchParamsRejectsTampering(t *testing.T) {
	values := url.Values{"vk_app_id": {"123"}, "vk_user_id": {"456"}, "vk_platform": {"android"}}
	values.Set("sign", signTestValues(values, "test-secret"))
	values.Set("vk_user_id", "789")
	if _, err := VerifyLaunchParams(values.Encode(), "123", "test-secret", time.Now()); err == nil {
		t.Fatal("tampered VK ID was accepted")
	}
}

func TestVerifyLaunchParamsRejectsWrongApp(t *testing.T) {
	values := url.Values{"vk_app_id": {"123"}, "vk_user_id": {"456"}}
	values.Set("sign", signTestValues(values, "test-secret"))
	if _, err := VerifyLaunchParams(values.Encode(), "999", "test-secret", time.Now()); err == nil {
		t.Fatal("launch params from another app were accepted")
	}
}

func TestVerifyLaunchParamsRejectsDuplicateValues(t *testing.T) {
	values := url.Values{"vk_app_id": {"123"}, "vk_user_id": {"456", "999"}, "sign": {"not-a-signature"}}
	if _, err := VerifyLaunchParams(values.Encode(), "123", "test-secret", time.Now()); err == nil {
		t.Fatal("duplicate query value was accepted")
	}
}

func TestVerifyLaunchParamsVKExample(t *testing.T) {
	// Example signature and protected key from the archived VKCOM implementation.
	// This guards against accidentally changing parameter canonicalization.
	const raw = "vk_user_id=494075&vk_app_id=6736218&vk_is_app_user=1&vk_are_notifications_enabled=1&vk_language=ru&vk_access_token_settings=&vk_platform=android&sign=htQFduJpLxz7ribXRZpDFUH-XEUhC9rBPTJkjUFEkRA"
	values, err := VerifyLaunchParams(raw, "6736218", "wvl68m4dR1UpLrVRli", time.Now())
	if err != nil {
		t.Fatalf("VK sample parameters rejected: %v", err)
	}
	if values.Get("vk_user_id") != "494075" {
		t.Fatalf("unexpected VK user id %q", values.Get("vk_user_id"))
	}
}
