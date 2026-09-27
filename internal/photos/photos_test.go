package photos

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testTransport struct {
	target string
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.URL.Scheme = "http"
	cloned.URL.Host = t.target
	return http.DefaultTransport.RoundTrip(cloned)
}

func newTestService(targetHost string) *Service {
	svc := NewService()
	svc.client.Transport = &testTransport{target: targetHost}
	return svc
}

func TestValidateAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid photos.app.goo.gl link",
			url:     "https://photos.app.goo.gl/abcdef123456",
			wantErr: false,
		},
		{
			name:    "valid photos.google.com link",
			url:     "https://photos.google.com/share/AF1QipNxyz?key=secretkey",
			wantErr: false,
		},
		{
			name:    "invalid scheme http",
			url:     "http://photos.google.com/share/AF1QipNxyz",
			wantErr: true,
		},
		{
			name:    "invalid unauthorized hostname",
			url:     "https://evil.com/share/AF1QipNxyz",
			wantErr: true,
		},
		{
			name:    "invalid subdomain attack",
			url:     "https://evil.photos.google.com/share/AF1QipNxyz",
			wantErr: true,
		},
		{
			name:    "empty url",
			url:     "",
			wantErr: true,
		},
		{
			name:    "relative url",
			url:     "/share/AF1QipNxyz",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ValidateAlbumURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateAlbumURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if !tt.wantErr && parsed == nil {
				t.Fatalf("expected non-nil URL for %q", tt.url)
			}
		})
	}
}

func TestNewService_RedirectLimit(t *testing.T) {
	redirectCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, fmt.Sprintf("/redirect-%d", redirectCount), http.StatusFound)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	_, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/loop")
	if err == nil {
		t.Fatal("expected error due to redirect limit, got nil")
	}
	if !strings.Contains(err.Error(), "stopped after 10 redirects") && !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("expected redirect limit error message, got: %v", err)
	}
}

func TestFetchPhotos_Success_And_VideoFiltering(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<head><title>Shared Album</title></head>
<body>
<script>
AF_initDataCallback({
  key: 'ds:1',
  hash: '2',
  data: [
    null,
    [
      ["photo-id-1", ["https://lh3.googleusercontent.com/pw/photo1_base", 1920, 1080]],
      ["video-id-1", ["https://lh3.googleusercontent.com/pw/video1_base", 1920, 1080], null, null, null, null, null, null, null, null, null, null, null, null, null, ["video/mp4"]],
      ["photo-id-2", ["https://lh3.googleusercontent.com/pw/photo2_base=w800-h600", 800, 600]]
    ]
  ],
  sideChannel: {}
});
</script>
</body>
</html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/album123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(photos) != 2 {
		t.Fatalf("expected 2 photos after video filtering, got %d: %+v", len(photos), photos)
	}

	if photos[0].ID != "photo-id-1" || !strings.HasSuffix(photos[0].DownloadURL, "=w0-h0") {
		t.Errorf("photo[0] unexpected: %+v", photos[0])
	}
	if photos[1].ID != "photo-id-2" || !strings.HasSuffix(photos[1].DownloadURL, "=w0-h0") {
		t.Errorf("photo[1] unexpected: %+v", photos[1])
	}
}

func TestFetchPhotos_ShortLinkRedirect(t *testing.T) {
	mockHTML := `
<!DOCTYPE html>
<html>
<body>
<script>
AF_initDataCallback({
  key: 'ds:1',
  data: [
    null,
    [
      ["photo-short-1", ["https://lh3.googleusercontent.com/pw/short_photo1", 1000, 1000]]
    ]
  ]
});
</script>
</body>
</html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/short" {
			http.Redirect(w, r, "/share/resolved-album", http.StatusFound)
			return
		}
		if r.URL.Path == "/share/resolved-album" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, err := svc.FetchPhotos(context.Background(), "https://photos.app.goo.gl/short")
	if err != nil {
		t.Fatalf("unexpected error following redirect: %v", err)
	}

	if len(photos) != 1 || photos[0].ID != "photo-short-1" {
		t.Fatalf("unexpected photos returned: %+v", photos)
	}
}

func TestFetchPhotos_RegexFallback(t *testing.T) {
	// HTML without AF_initDataCallback, containing direct lh3 URLs
	mockHTML := `
<!DOCTYPE html>
<html>
<body>
  <img src="https://lh3.googleusercontent.com/pw/AP1GczPhotoAlpha=w500" />
  <img src="https://lh3.googleusercontent.com/pw/AP1GczPhotoBeta=d" />
</body>
</html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/fallback")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(photos) != 2 {
		t.Fatalf("expected 2 photos via regex extraction, got %d: %+v", len(photos), photos)
	}
}

func TestFetchPhotos_EmptyAlbum(t *testing.T) {
	mockHTML := `<!DOCTYPE html><html><body><p>Empty Album</p></body></html>`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockHTML))
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/empty")
	if err != nil {
		t.Fatalf("expected no error for empty album, got: %v", err)
	}
	if len(photos) != 0 {
		t.Fatalf("expected 0 photos, got %d", len(photos))
	}
}

func TestFetchPhotos_ErrorStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		checkErr   func(err error) bool
	}{
		{
			name:       "HTTP 429 Rate Limit",
			statusCode: http.StatusTooManyRequests,
			checkErr: func(err error) bool {
				return errors.Is(err, ErrRateLimited) || strings.Contains(err.Error(), "429")
			},
		},
		{
			name:       "HTTP 404 Not Found",
			statusCode: http.StatusNotFound,
			checkErr: func(err error) bool {
				return strings.Contains(err.Error(), "404")
			},
		},
		{
			name:       "HTTP 403 Forbidden",
			statusCode: http.StatusForbidden,
			checkErr: func(err error) bool {
				return strings.Contains(err.Error(), "403")
			},
		},
		{
			name:       "HTTP 500 Server Error",
			statusCode: http.StatusInternalServerError,
			checkErr: func(err error) bool {
				return strings.Contains(err.Error(), "500")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer ts.Close()

			svc := newTestService(ts.Listener.Addr().String())
			_, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/test")
			if err == nil {
				t.Fatalf("expected error for status code %d, got nil", tt.statusCode)
			}
			if !tt.checkErr(err) {
				t.Fatalf("unexpected error response for status code %d: %v", tt.statusCode, err)
			}
		})
	}
}

func TestDownloadImage_SuccessAndFailures(t *testing.T) {
	expectedBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ok.jpg" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expectedBytes)
			return
		}
		if r.URL.Path == "/slow.jpg" {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expectedBytes)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := NewService()

	// 1. Success case
	img, err := svc.DownloadImage(context.Background(), ts.URL+"/ok.jpg")
	if err != nil {
		t.Fatalf("expected successful image download, got: %v", err)
	}
	if string(img) != string(expectedBytes) {
		t.Fatalf("downloaded bytes mismatch: got %v, want %v", img, expectedBytes)
	}

	// 2. HTTP 404 Failure case
	_, err = svc.DownloadImage(context.Background(), ts.URL+"/missing.jpg")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	// 3. Context cancelled / timeout failure case
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = svc.DownloadImage(ctx, ts.URL+"/slow.jpg")
	if err == nil {
		t.Fatal("expected error on cancelled context, got nil")
	}
}
