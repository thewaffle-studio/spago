package spago

import (
	"embed"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type SPAServer struct {
	fs        fs.FS
	dirPath   string
	useFS     bool
	basePath  string
	entryPath string
}

func NewFromFS(contentFS embed.FS, subPath string) (*SPAServer, error) {
	subFS, err := fs.Sub(contentFS, subPath)
	if err != nil {
		return nil, err
	}

	return &SPAServer{
		fs:        subFS,
		useFS:     true,
		basePath:  "/",
		entryPath: "index.html",
	}, nil
}

func NewFromDir(dirPath string) *SPAServer {
	return &SPAServer{
		dirPath:   dirPath,
		useFS:     false,
		basePath:  "/",
		entryPath: "index.html",
	}
}

func (s *SPAServer) WithBasePath(path string) *SPAServer {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") && path != "/" {
		path = path + "/"
	}
	s.basePath = path

	return s
}

func (s *SPAServer) WithEntryFile(entryPath string) *SPAServer {
	s.entryPath = entryPath
	return s
}

func (s *SPAServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, s.basePath) {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, s.basePath)
		path = strings.TrimPrefix(path, "/")

		if path == "" {
			s.serveEntry(w, r)
			return
		}

		exists := s.fileExists(path)

		if exists {
			if strings.HasSuffix(path, ".js") {
				w.Header().Set("Content-Type", "application/javascript")
			}

			s.serveFile(w, r, path)
			return
		}

		s.serveEntry(w, r)
	})
}

func (s *SPAServer) fileExists(path string) bool {
	if s.useFS {
		_, err := fs.Stat(s.fs, path)
		return err == nil
	} else {
		fullPath := filepath.Join(s.dirPath, path)
		_, err := os.Stat(fullPath)
		return err == nil
	}
}

func (s *SPAServer) serveEntry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if s.useFS {
		content, err := fs.ReadFile(s.fs, s.entryPath)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Write(content)
	} else {
		http.ServeFile(w, r, filepath.Join(s.dirPath, s.entryPath))
	}
}

func (s *SPAServer) serveFile(w http.ResponseWriter, r *http.Request, path string) {
	if s.useFS {
		fileServer := http.FileServer(http.FS(s.fs))
		r2 := cloneRequest(r)
		r2.URL.Path = "/" + path
		fileServer.ServeHTTP(w, r2)
	}
}

func cloneRequest(r *http.Request) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	return r2
}
