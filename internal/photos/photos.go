package photos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ErrRateLimited is returned when Google Photos responds with HTTP 429 Too Many Requests.
var ErrRateLimited = errors.New("rate limited by Google Photos (HTTP 429)")

// Photo represents a media item extracted from a Google Photos shared album.
type Photo struct {
	ID          string
	DownloadURL string
}

// Service handles public Google Photos shared album parsing and media downloads.
type Service struct {
	client *http.Client
}

// NewService creates a new Service configured with a 30-second client timeout
// and redirect check supporting up to 10 redirects.
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
	}
}

// NewServiceWithClient creates a new Service with a custom HTTP client,
// enabling custom transports and mock servers in integration tests.
func NewServiceWithClient(client *http.Client) *Service {
	return &Service{
		client: client,
	}
}


// ValidateAlbumURL enforces https scheme and restricts allowed hostnames
// strictly to photos.app.goo.gl and photos.google.com.
func ValidateAlbumURL(urlStr string) (*url.URL, error) {
	if strings.TrimSpace(urlStr) == "" {
		return nil, fmt.Errorf("album URL cannot be empty")
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("malformed album URL: %w", err)
	}

	if u.Scheme != "https" {
		return nil, fmt.Errorf("invalid scheme %q: only https is allowed", u.Scheme)
	}

	host := u.Hostname()
	if host != "photos.app.goo.gl" && host != "photos.google.com" {
		return nil, fmt.Errorf("unauthorized host %q: only photos.app.goo.gl and photos.google.com are allowed", host)
	}

	return u, nil
}

// makeDownloadURL constructs a high-resolution direct download URL by appending =w0-h0 or preserving =d.
func makeDownloadURL(base string) string {
	if strings.HasSuffix(base, "=d") || strings.HasSuffix(base, "=w0-h0") {
		return base
	}
	if idx := strings.LastIndex(base, "="); idx != -1 {
		base = base[:idx]
	}
	return base + "=w0-h0"
}

// isVideoStructure checks whether a data structure contains video metadata or indicators.
func isVideoStructure(v any) bool {
	switch val := v.(type) {
	case string:
		lower := strings.ToLower(val)
		if strings.Contains(lower, "video") {
			return true
		}
	case []any:
		for _, elem := range val {
			if isVideoStructure(elem) {
				return true
			}
		}
	case map[string]any:
		for k, elem := range val {
			if strings.Contains(strings.ToLower(k), "video") {
				return true
			}
			if isVideoStructure(elem) {
				return true
			}
		}
	}
	return false
}

// extractBalanced scans a string starting with an open bracket or brace and returns the balanced enclosed substring.
func extractBalanced(s string) (string, bool) {
	if len(s) == 0 {
		return "", false
	}
	openChar := s[0]
	var closeChar byte
	if openChar == '[' {
		closeChar = ']'
	} else if openChar == '{' {
		closeChar = '}'
	} else {
		return "", false
	}

	depth := 0
	inString := false
	var quoteChar byte
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == quoteChar {
				inString = false
			}
			continue
		}

		if c == '"' || c == '\'' {
			inString = true
			quoteChar = c
			continue
		}

		if c == openChar {
			depth++
		} else if c == closeChar {
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}

// extractPhotosFromData recursively traverses unmarshaled AF_initDataCallback payload to find photo entries.
func extractPhotosFromData(raw any) []Photo {
	var photos []Photo
	seen := make(map[string]bool)

	var walk func(v any)
	walk = func(v any) {
		switch node := v.(type) {
		case []any:
			// Check if this slice represents a media item: [id, [url, width, height], ...]
			if len(node) >= 2 {
				id, hasID := node[0].(string)
				var imgURL string

				// Check if node[1] is a slice containing the URL
				if sub, ok := node[1].([]any); ok && len(sub) > 0 {
					if u, ok := sub[0].(string); ok && strings.HasPrefix(u, "https://lh3.googleusercontent.com/") {
						imgURL = u
					}
				} else if u, ok := node[1].(string); ok && strings.HasPrefix(u, "https://lh3.googleusercontent.com/") {
					imgURL = u
				}

				if imgURL != "" && hasID && id != "" {
					// Check if remaining elements indicate a video
					isVideo := false
					for i := 2; i < len(node); i++ {
						if isVideoStructure(node[i]) {
							isVideo = true
							break
						}
					}

					if !isVideo && !seen[id] {
						seen[id] = true
						photos = append(photos, Photo{
							ID:          id,
							DownloadURL: makeDownloadURL(imgURL),
						})
						return
					} else if isVideo {
						return
					}
				}
			}

			for _, elem := range node {
				walk(elem)
			}

		case map[string]any:
			if isVideoStructure(node) {
				return
			}
			for _, elem := range node {
				walk(elem)
			}
		}
	}

	walk(raw)
	return photos
}

// extractPhotosRegex scans HTML for Google Photos lh3 image URLs and extracts photo IDs and download URLs.
func extractPhotosRegex(html string) []Photo {
	var photos []Photo
	seen := make(map[string]bool)

	// Match https://lh3.googleusercontent.com/pw/<slug> or https://lh3.googleusercontent.com/<slug>
	re := regexp.MustCompile(`https://lh3\.googleusercontent\.com/(?:pw/)?([a-zA-Z0-9_\-]+)(?:=[a-zA-Z0-9_\-]+)*`)
	matches := re.FindAllStringSubmatch(html, -1)

	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		fullURL := m[0]
		id := m[1]

		if seen[id] {
			continue
		}
		seen[id] = true

		photos = append(photos, Photo{
			ID:          id,
			DownloadURL: makeDownloadURL(fullURL),
		})
	}

	return photos
}

// FetchPhotos fetches a Google Photos shared album page, parses media metadata,
// filters out video items, and returns high-resolution Photo records.
func (s *Service) FetchPhotos(ctx context.Context, albumURL string) ([]Photo, error) {
	if _, err := ValidateAlbumURL(albumURL); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, albumURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	client := s.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch album page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, ErrRateLimited
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("album not found (HTTP 404): %s", albumURL)
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("album forbidden (HTTP 403): %s", albumURL)
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("google photos server error (HTTP %d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d fetching album", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read album response body: %w", err)
	}

	html := string(bodyBytes)

	// 1. Try parsing AF_initDataCallback
	var parsedPhotos []Photo
	idx := 0
	for {
		pos := strings.Index(html[idx:], "AF_initDataCallback")
		if pos == -1 {
			break
		}
		start := idx + pos
		dataIdx := strings.Index(html[start:], "data:")
		if dataIdx != -1 {
			sub := html[start+dataIdx+5:]
			trimmed := strings.TrimLeft(sub, " \t\r\n")
			if len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
				if jsonStr, ok := extractBalanced(trimmed); ok {
					var raw any
					if err := json.Unmarshal([]byte(jsonStr), &raw); err == nil {
						extracted := extractPhotosFromData(raw)
						if len(extracted) > 0 {
							parsedPhotos = append(parsedPhotos, extracted...)
						}
					}
				}
			}
		}
		idx = start + len("AF_initDataCallback")
	}

	if len(parsedPhotos) > 0 {
		return parsedPhotos, nil
	}

	// 2. Fallback to regex extraction
	regexPhotos := extractPhotosRegex(html)
	if len(regexPhotos) > 0 {
		return regexPhotos, nil
	}

	// Return empty slice if no photos found in valid HTML
	return []Photo{}, nil
}

// DownloadImage streams image binary bytes from downloadURL with a 60-second context timeout.
func (s *Service) DownloadImage(ctx context.Context, downloadURL string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create image request: %w", err)
	}

	client := s.client
	if client == nil {
		client = http.DefaultClient
	} else if client.Timeout > 0 && client.Timeout < 60*time.Second {
		c := *client
		c.Timeout = 0 // use request context 60-second timeout
		client = &c
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status downloading image: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image body: %w", err)
	}

	return data, nil
}
