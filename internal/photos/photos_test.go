package photos

import (
	"context"
	"encoding/json"
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

func TestExtractContinuationToken_Success(t *testing.T) {
	// Direct array payload
	rawJSON := `[
		null,
		[
			["photo-id-1", ["https://lh3.googleusercontent.com/pw/photo1_base", 1920, 1080]]
		],
		"token-abc-123"
	]`
	var raw any
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}

	token := extractContinuationToken(raw)
	if token != "token-abc-123" {
		t.Fatalf("expected token %q, got %q", "token-abc-123", token)
	}

	// Wrapped payload in callback map
	wrappedJSON := `{
		"key": "ds:1",
		"data": [
			null,
			[
				["photo-id-2", ["https://lh3.googleusercontent.com/pw/photo2_base", 800, 600]]
			],
			"token-wrapped-456"
		]
	}`
	var wrapped any
	if err := json.Unmarshal([]byte(wrappedJSON), &wrapped); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}
	tokenWrapped := extractContinuationToken(wrapped)
	if tokenWrapped != "token-wrapped-456" {
		t.Fatalf("expected token %q, got %q", "token-wrapped-456", tokenWrapped)
	}
}

func TestExtractContinuationToken_Missing(t *testing.T) {
	// Case 1: Array without token element
	rawJSON := `[
		null,
		[
			["photo-id-1", ["https://lh3.googleusercontent.com/pw/photo1_base", 1920, 1080]]
		]
	]`
	var raw any
	if err := json.Unmarshal([]byte(rawJSON), &raw); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}
	token := extractContinuationToken(raw)
	if token != "" {
		t.Fatalf("expected empty token, got %q", token)
	}

	// Case 2: Array with null token element
	rawNullJSON := `[
		null,
		[
			["photo-id-1", ["https://lh3.googleusercontent.com/pw/photo1_base", 1920, 1080]]
		],
		null
	]`
	var rawNull any
	if err := json.Unmarshal([]byte(rawNullJSON), &rawNull); err != nil {
		t.Fatalf("failed to unmarshal test JSON: %v", err)
	}
	tokenNull := extractContinuationToken(rawNull)
	if tokenNull != "" {
		t.Fatalf("expected empty token for null, got %q", tokenNull)
	}
}

func TestFetchContinuationBatch_Success(t *testing.T) {
	mockBatchJSON := `[
		null,
		[
			["photo-batch-1", ["https://lh3.googleusercontent.com/pw/photo_batch1_base", 1920, 1080]],
			["video-batch-1", ["https://lh3.googleusercontent.com/pw/video_batch1_base", 1920, 1080], null, null, null, null, null, null, null, null, null, null, null, null, null, ["video/mp4"]],
			["photo-batch-2", ["https://lh3.googleusercontent.com/pw/photo_batch2_base=w800-h600", 800, 600]]
		],
		"next-token-789"
	]`
	envelopeJSON, err := json.Marshal([][]any{
		{"wrb.fr", albumContinuationRPCID, mockBatchJSON, nil, nil, nil, "generic"},
	})
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}
	mockResponse := ")]}'\n\n" + string(envelopeJSON)

	receivedReq := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = true
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.URL.Path != batchExecutePath {
			t.Errorf("expected path %s, got %s", batchExecutePath, r.URL.Path)
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "Mozilla/5.0") {
			t.Errorf("expected browser User-Agent, got %s", r.Header.Get("User-Agent"))
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("failed to parse form body: %v", err)
		}
		reqVal := r.PostForm.Get("f.req")
		if !strings.Contains(reqVal, albumContinuationRPCID) || !strings.Contains(reqVal, "token-start") {
			t.Errorf("f.req missing rpcid or token: %s", reqVal)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, nextToken, err := svc.fetchContinuationBatch(context.Background(), "photos.google.com", "token-start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !receivedReq {
		t.Fatal("expected mock server to receive request")
	}

	if len(photos) != 2 {
		t.Fatalf("expected 2 photos (video filtered), got %d: %+v", len(photos), photos)
	}
	if photos[0].ID != "photo-batch-1" || !strings.HasSuffix(photos[0].DownloadURL, "=w0-h0") {
		t.Errorf("unexpected photo[0]: %+v", photos[0])
	}
	if photos[1].ID != "photo-batch-2" || !strings.HasSuffix(photos[1].DownloadURL, "=w0-h0") {
		t.Errorf("unexpected photo[1]: %+v", photos[1])
	}
	if nextToken != "next-token-789" {
		t.Errorf("expected next token %q, got %q", "next-token-789", nextToken)
	}
}

func TestFetchContinuationBatch_RateLimited(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, nextToken, err := svc.fetchContinuationBatch(context.Background(), "photos.google.com", "token-ratelimit")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
	if len(photos) != 0 || nextToken != "" {
		t.Fatalf("expected empty return on rate limit, got photos=%v, token=%q", photos, nextToken)
	}
}

func TestFetchContinuationBatch_MalformedPayload(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
	}{
		{
			name:         "corrupted XSSI guard prefix",
			responseBody: "CORRUPTED_PREFIX\n[[1, 2, 3]]",
		},
		{
			name:         "malformed JSON after valid prefix",
			responseBody: ")]}'\n\n{invalid json content",
		},
		{
			name:         "empty body without XSSI prefix",
			responseBody: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer ts.Close()

			svc := newTestService(ts.Listener.Addr().String())
			photos, nextToken, err := svc.fetchContinuationBatch(context.Background(), "photos.google.com", "token-malformed")
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tt.name)
			}
			if len(photos) != 0 || nextToken != "" {
				t.Fatalf("expected empty photos and token on error, got photos=%v, token=%q", photos, nextToken)
			}
		})
	}
}

func TestFetchContinuationBatch_EmptyBatch(t *testing.T) {
	emptyBatchJSON := `[null, [], null]`
	envelopeJSON, err := json.Marshal([][]any{
		{"wrb.fr", albumContinuationRPCID, emptyBatchJSON, nil, nil, nil, "generic"},
	})
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}
	mockResponse := ")]}'\n\n" + string(envelopeJSON)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(mockResponse))
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, nextToken, err := svc.fetchContinuationBatch(context.Background(), "photos.google.com", "token-empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if photos == nil || len(photos) != 0 {
		t.Fatalf("expected empty non-nil slice, got: %+v", photos)
	}
	if nextToken != "" {
		t.Fatalf("expected empty token, got: %q", nextToken)
	}
}

func makeTestAlbumHTML(photos []Photo, initialToken string) string {
	var items [][]any
	for _, p := range photos {
		items = append(items, []any{p.ID, []any{p.DownloadURL, 1920, 1080}})
	}
	var data []any
	if initialToken != "" {
		data = []any{nil, items, initialToken}
	} else {
		data = []any{nil, items}
	}
	dataJSON, _ := json.Marshal(data)
	return fmt.Sprintf(`<!DOCTYPE html><html><body><script>AF_initDataCallback({key:'ds:1',data:%s});</script></body></html>`, string(dataJSON))
}

func makeTestBatchResponse(photos []Photo, nextToken string) string {
	var items [][]any
	for _, p := range photos {
		items = append(items, []any{p.ID, []any{p.DownloadURL, 1920, 1080}})
	}
	var batchData []any
	if nextToken != "" {
		batchData = []any{nil, items, nextToken}
	} else {
		batchData = []any{nil, items}
	}
	batchJSON, _ := json.Marshal(batchData)
	envelope, _ := json.Marshal([][]any{
		{"wrb.fr", albumContinuationRPCID, string(batchJSON), nil, nil, nil, "generic"},
	})
	return ")]}'\n\n" + string(envelope)
}

func makeTestBatchResponseWithVideos(photos []Photo, videoIDs []string, nextToken string) string {
	var items [][]any
	for _, p := range photos {
		items = append(items, []any{p.ID, []any{p.DownloadURL, 1920, 1080}})
	}
	for _, vid := range videoIDs {
		items = append(items, []any{vid, []any{"https://lh3.googleusercontent.com/pw/video", 1920, 1080}, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, []any{"video/mp4"}})
	}
	var batchData []any
	if nextToken != "" {
		batchData = []any{nil, items, nextToken}
	} else {
		batchData = []any{nil, items}
	}
	batchJSON, _ := json.Marshal(batchData)
	envelope, _ := json.Marshal([][]any{
		{"wrb.fr", albumContinuationRPCID, string(batchJSON), nil, nil, nil, "generic"},
	})
	return ")]}'\n\n" + string(envelope)
}

func TestFetchPhotos_Pagination_Success(t *testing.T) {
	initialPhotos := make([]Photo, 200)
	for i := 0; i < 200; i++ {
		initialPhotos[i] = Photo{
			ID:          fmt.Sprintf("photo-init-%03d", i),
			DownloadURL: fmt.Sprintf("https://lh3.googleusercontent.com/pw/init_%03d=w0-h0", i),
		}
	}
	mockHTML := makeTestAlbumHTML(initialPhotos, "token-batch-1")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			_ = r.ParseForm()
			reqVal := r.PostForm.Get("f.req")
			var batchNum int
			if strings.Contains(reqVal, "token-batch-1") {
				batchNum = 1
			} else if strings.Contains(reqVal, "token-batch-2") {
				batchNum = 2
			} else if strings.Contains(reqVal, "token-batch-3") {
				batchNum = 3
			} else if strings.Contains(reqVal, "token-batch-4") {
				batchNum = 4
			} else if strings.Contains(reqVal, "token-batch-5") {
				batchNum = 5
			}

			if batchNum > 0 {
				batchPhotos := make([]Photo, 200)
				for i := 0; i < 200; i++ {
					idx := (batchNum * 200) + i
					batchPhotos[i] = Photo{
						ID:          fmt.Sprintf("photo-batch-%04d", idx),
						DownloadURL: fmt.Sprintf("https://lh3.googleusercontent.com/pw/batch_%04d=w0-h0", idx),
					}
				}
				var nextTok string
				if batchNum < 5 {
					nextTok = fmt.Sprintf("token-batch-%d", batchNum+1)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(makeTestBatchResponse(batchPhotos, nextTok)))
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	photos, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/large-album")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(photos) < 1000 {
		t.Fatalf("expected >= 1000 photos, got %d", len(photos))
	}
	if len(photos) != 1200 {
		t.Fatalf("expected 1200 photos, got %d", len(photos))
	}

	seen := make(map[string]bool)
	for _, p := range photos {
		if seen[p.ID] {
			t.Fatalf("duplicate photo ID found: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestFetchPhotos_SinglePage_NoToken(t *testing.T) {
	photos := []Photo{
		{ID: "p1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
		{ID: "p2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(photos, "")

	batchCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.URL.Path == batchExecutePath {
			batchCount++
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/single-page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(result))
	}
	if batchCount != 0 {
		t.Fatalf("expected 0 batchexecute requests, got %d", batchCount)
	}
}

func TestFetchPhotos_CrossPageDeduplication(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
		{ID: "photo-2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-dedup")

	page2 := []Photo{
		{ID: "photo-2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"},
		{ID: "photo-3", DownloadURL: "https://lh3.googleusercontent.com/pw/p3=w0-h0"},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(makeTestBatchResponse(page2, "")))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/dedup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 unique photos, got %d: %+v", len(result), result)
	}
	expectedIDs := []string{"photo-1", "photo-2", "photo-3"}
	for i, id := range expectedIDs {
		if result[i].ID != id {
			t.Errorf("expected photo[%d].ID=%q, got %q", i, id, result[i].ID)
		}
	}
}

func TestFetchPhotos_VideoFilteringAcrossPages(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-video")

	page2 := []Photo{
		{ID: "photo-2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"},
	}
	videoIDs := []string{"video-batch-1"}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(makeTestBatchResponseWithVideos(page2, videoIDs, "")))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/video-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 photos, got %d: %+v", len(result), result)
	}
	for _, p := range result {
		if strings.Contains(p.ID, "video") {
			t.Errorf("found video in photos result: %s", p.ID)
		}
	}
}

func TestFetchPhotos_CyclicTokenDetection(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-cycle")

	page2 := []Photo{
		{ID: "photo-2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"},
	}

	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			callCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			// Return identical token "token-cycle" causing potential loop
			_, _ = w.Write([]byte(makeTestBatchResponse(page2, "token-cycle")))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/cyclic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(result))
	}
	if callCount != 1 {
		t.Fatalf("expected exactly 1 continuation call before cycle break, got %d", callCount)
	}
}

func TestFetchPhotos_MaxPageSafetyLimit(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "tok-0")

	callCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			callCount++
			nextTok := fmt.Sprintf("tok-%d", callCount)
			batchPhotos := []Photo{
				{ID: fmt.Sprintf("photo-inf-%d", callCount), DownloadURL: "https://lh3.googleusercontent.com/pw/inf=w0-h0"},
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(makeTestBatchResponse(batchPhotos, nextTok)))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/infinite")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != MaxPaginationPages {
		t.Fatalf("expected exactly %d continuation calls (MaxPaginationPages), got %d", MaxPaginationPages, callCount)
	}
	if len(result) != 1+MaxPaginationPages {
		t.Fatalf("expected %d photos, got %d", 1+MaxPaginationPages, len(result))
	}
}

func TestFetchPhotos_RateLimitMidPagination(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-rl-mid")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	_, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/ratelimit")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got: %v", err)
	}
}

func TestFetchPhotos_ContextCancellationMidPagination(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-cancel")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			cancel() // cancel context as soon as batch execute is called
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(makeTestBatchResponse([]Photo{{ID: "photo-2", DownloadURL: "https://lh3.googleusercontent.com/pw/p2=w0-h0"}}, "next-token")))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	_, err := svc.FetchPhotos(ctx, "https://photos.google.com/share/cancel")
	if err == nil {
		t.Fatal("expected context error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}
}

func TestFetchPhotos_EmptyContinuationBatch(t *testing.T) {
	page1 := []Photo{
		{ID: "photo-1", DownloadURL: "https://lh3.googleusercontent.com/pw/p1=w0-h0"},
	}
	mockHTML := makeTestAlbumHTML(page1, "token-empty-batch")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(mockHTML))
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == batchExecutePath {
			emptyBatchJSON := `[null, [], null]`
			envelopeJSON, _ := json.Marshal([][]any{
				{"wrb.fr", albumContinuationRPCID, emptyBatchJSON, nil, nil, nil, "generic"},
			})
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(")]}'\n\n" + string(envelopeJSON)))
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	svc := newTestService(ts.Listener.Addr().String())
	result, err := svc.FetchPhotos(context.Background(), "https://photos.google.com/share/empty-batch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 photo from page 1, got %d", len(result))
	}
	if result[0].ID != "photo-1" {
		t.Fatalf("expected photo ID %q, got %q", "photo-1", result[0].ID)
	}
}


