package proxy

import (
	"net/http"
	"strings"
)

// ExtractMetadata performs shallow DPI on an HTTP request, returning
// a map of interesting fields suitable for pipeline event metadata.
func ExtractMetadata(r *http.Request, proto Protocol) map[string]any {
	m := map[string]any{
		"method":   r.Method,
		"host":     r.Host,
		"url":      r.URL.String(),
		"protocol": string(proto),
	}

	if ua := r.Header.Get("User-Agent"); ua != "" {
		m["user_agent"] = ua
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		m["content_type"] = ct
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		m["referer"] = ref
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		// Log presence but not the value
		m["has_auth"] = true
		m["auth_scheme"] = strings.SplitN(auth, " ", 2)[0]
	}
	if r.ContentLength > 0 {
		m["content_length"] = r.ContentLength
	}

	// Flag potentially suspicious patterns
	if isSuspicious(r) {
		m["suspicious"] = true
	}

	return m
}

// isSuspicious flags basic red-flag heuristics on the request.
func isSuspicious(r *http.Request) bool {
	url := strings.ToLower(r.URL.String())
	// Very basic heuristic set — expand with threat intel feeds later
	patterns := []string{
		"/etc/passwd",
		"/etc/shadow",
		"..%2f",
		"../../",
		"<script",
		"cmd.exe",
		"powershell",
		"/wp-admin",
		"/phpmyadmin",
	}
	for _, p := range patterns {
		if strings.Contains(url, p) {
			return true
		}
	}
	return false
}
