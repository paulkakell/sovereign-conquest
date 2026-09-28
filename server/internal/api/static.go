package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// spaHandler confines every lookup to WEB_ROOT, including directory indexes and
// the SPA fallback. os.Root also prevents symlink escapes during concurrent file
// replacement; cleaning or checking a string prefix alone cannot provide that.
func spaHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		if strings.ContainsAny(r.URL.Path, "\\\x00") {
			http.NotFound(w, r)
			return
		}
		for _, component := range strings.Split(r.URL.Path, "/") {
			if component == ".." {
				http.NotFound(w, r)
				return
			}
		}

		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "."
		}
		if name == "api" || strings.HasPrefix(name, "api/") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		if !fs.ValidPath(name) || !filepath.IsLocal(name) {
			http.NotFound(w, r)
			return
		}

		webRoot, err := os.OpenRoot(root)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = webRoot.Close() }()

		file, info, err := openWebFile(webRoot, name)
		fallback := false
		if err != nil {
			// Only genuinely missing extensionless routes use the SPA fallback.
			// Permission errors, symlink escapes and special files fail closed.
			if !errors.Is(err, fs.ErrNotExist) || path.Ext(name) != "" {
				http.NotFound(w, r)
				return
			}
			file, info, err = openWebFile(webRoot, "index.html")
			fallback = true
		}
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = file.Close() }()

		if fallback || path.Ext(info.Name()) == ".html" {
			w.Header().Set("Cache-Control", "no-store")
		}
		if strings.HasSuffix(r.URL.Path, "/index.html") {
			w.Header().Set("Location", "./")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		// Serve the validated open descriptor instead of reopening a path. This
		// retains HEAD, Range and conditional requests without directory listings.
		http.ServeContent(w, r, info.Name(), info.ModTime(), file)
	}
}

func openWebFile(root *os.Root, name string) (*os.File, fs.FileInfo, error) {
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}
	if info.IsDir() {
		_ = file.Close()
		file, err = root.Open(path.Join(name, "index.html"))
		if err != nil {
			return nil, nil, err
		}
		info, err = file.Stat()
		if err != nil {
			_ = file.Close()
			return nil, nil, err
		}
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, nil, fs.ErrPermission
	}
	return file, info, nil
}
