package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func staticFixture(t testing.TB) (string, string) {
	t.Helper()
	base := t.TempDir()
	root := filepath.Join(base, "web")
	outside := filepath.Join(base, "private")
	for _, dir := range []string{root, outside, filepath.Join(root, "nested"), filepath.Join(root, "empty")} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range map[string]string{
		filepath.Join(root, "index.html"):           "<html>home</html>",
		filepath.Join(root, "app.js"):               "console.log('safe');",
		filepath.Join(root, "nested", "index.html"): "<html>nested</html>",
		filepath.Join(outside, "secret.txt"):        "outside-private-fixture",
		filepath.Join(outside, "index.html"):        "outside-private-fixture",
	} {
		if err := os.WriteFile(name, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root, outside
}

func TestSPAHandlerRejectsFilesystemEscapes(t *testing.T) {
	root, outside := staticFixture(t)
	for name, target := range map[string]string{
		"relative.txt": "../private/secret.txt",
		"absolute.txt": filepath.Join(outside, "secret.txt"),
		"linked-dir":   "../private",
	} {
		if err := os.Symlink(target, filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink("../../private/index.html", filepath.Join(root, "empty", "index.html")); err != nil {
		t.Fatal(err)
	}
	handler := spaHandler(root)
	for _, target := range []string{
		"/relative.txt", "/absolute.txt", "/linked-dir/secret.txt", "/linked-dir/", "/empty/",
		"/../private/secret.txt", "/%2e%2e/private/secret.txt", "/nested/../../private/secret.txt",
		"/..%5cprivate%5csecret.txt", "/%00secret.txt",
	} {
		for _, method := range []string{http.MethodGet, http.MethodHead} {
			t.Run(method+target, func(t *testing.T) {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(method, target, nil))
				if response.Code != http.StatusNotFound {
					t.Fatalf("status=%d want 404", response.Code)
				}
				for _, secret := range []string{"outside-private-fixture", outside, root} {
					if strings.Contains(response.Body.String(), secret) {
						t.Fatalf("response exposed private data: %q", response.Body.String())
					}
				}
			})
		}
	}
}

func TestSPAHandlerRejectsEscapingFallbackIndex(t *testing.T) {
	root, outside := staticFixture(t)
	if err := os.Remove(filepath.Join(root, "index.html")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "index.html"), filepath.Join(root, "index.html")); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"/", "/client/route"} {
		response := httptest.NewRecorder()
		spaHandler(root).ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "outside-private-fixture") {
			t.Fatalf("unsafe index fallback for %s: status=%d body=%q", target, response.Code, response.Body.String())
		}
	}
}

func TestSPAHandlerRejectsSymlinkReplacementRace(t *testing.T) {
	root, _ := staticFixture(t)
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		for {
			for _, target := range []string{"app.js", "../private/secret.txt"} {
				select {
				case <-stop:
					done <- nil
					return
				default:
				}
				if err := os.Symlink(target, filepath.Join(root, "next.js")); err != nil {
					done <- err
					return
				}
				if err := os.Rename(filepath.Join(root, "next.js"), filepath.Join(root, "changing.js")); err != nil {
					done <- err
					return
				}
			}
		}
	}()
	defer func() {
		close(stop)
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	handler := spaHandler(root)
	for range 200 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/changing.js", nil))
		if response.Code != http.StatusOK && response.Code != http.StatusNotFound {
			t.Fatalf("unexpected status %d", response.Code)
		}
		if response.Code == http.StatusOK && response.Body.String() != "console.log('safe');" {
			t.Fatalf("file replacement escaped root: %q", response.Body.String())
		}
	}
}

func TestSPAHandlerPreservesWebBehavior(t *testing.T) {
	root, _ := staticFixture(t)
	if err := os.Symlink("app.js", filepath.Join(root, "alias.js")); err != nil {
		t.Fatal(err)
	}
	handler := spaHandler(root)
	for _, tc := range []struct {
		method, target string
		status         int
		body, cache    string
	}{
		{"GET", "/", 200, "<html>home</html>", "no-store"},
		{"GET", "/client/route", 200, "<html>home</html>", "no-store"},
		{"GET", "/app.js", 200, "console.log('safe');", ""},
		{"GET", "/alias.js", 200, "console.log('safe');", ""},
		{"HEAD", "/app.js", 200, "", ""},
		{"HEAD", "/", 200, "", "no-store"},
		{"GET", "/nested/", 200, "<html>nested</html>", "no-store"},
		{"GET", "/empty/", 200, "<html>home</html>", "no-store"},
		{"GET", "/missing.js", 404, "404 page not found\n", ""},
		{"GET", "/api", 404, "{\"error\":\"not found\",\"ok\":false}\n", ""},
		{"GET", "/api/unknown", 404, "{\"error\":\"not found\",\"ok\":false}\n", ""},
		{"POST", "/", 404, "404 page not found\n", ""},
	} {
		t.Run(tc.method+tc.target, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(tc.method, tc.target, nil))
			if response.Code != tc.status || response.Body.String() != tc.body || response.Header().Get("Cache-Control") != tc.cache {
				t.Fatalf("status=%d body=%q cache=%q", response.Code, response.Body.String(), response.Header().Get("Cache-Control"))
			}
		})
	}
	t.Run("index redirect", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/nested/index.html", nil))
		if response.Code != http.StatusMovedPermanently || response.Header().Get("Location") != "./" {
			t.Fatalf("status=%d location=%q", response.Code, response.Header().Get("Location"))
		}
	})
	t.Run("range", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		request.Header.Set("Range", "bytes=0-6")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusPartialContent || response.Body.String() != "console" {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
	})
	t.Run("conditional", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		request.Header.Set("If-Modified-Since", time.Now().Add(time.Hour).UTC().Format(http.TimeFormat))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotModified || response.Body.Len() != 0 {
			t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
		}
	})
}

func BenchmarkSPAHandler(b *testing.B) {
	root, _ := staticFixture(b)
	handler := spaHandler(root)
	request := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			b.Fatal(response.Code)
		}
	}
}
