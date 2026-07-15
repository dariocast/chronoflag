package httpapi

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
)

//go:embed all:dist
var webAssets embed.FS

func staticHandler() http.Handler {
	root, e := fs.Sub(webAssets, "dist")
	if e != nil {
		panic(e)
	}
	files := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if clean == "." {
			clean = "index.html"
		}
		if _, e := fs.Stat(root, clean); e != nil {
			clean = "index.html"
		}
		if strings.Contains(clean, "/_app/immutable/") {
			w.Header().Set("Cache-Control", "public,max-age=31536000,immutable")
		} else {
			w.Header().Set("Cache-Control", "no-store")
		}
		if ext := path.Ext(clean); ext != "" {
			if kind := mime.TypeByExtension(ext); kind != "" {
				w.Header().Set("Content-Type", kind)
			}
		}
		if clean == "index.html" {
			body, err := fs.ReadFile(root, clean)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			nonceBytes := make([]byte, 18)
			if _, err = rand.Read(nonceBytes); err != nil {
				http.Error(w, "entropy unavailable", http.StatusInternalServerError)
				return
			}
			nonce := base64.RawStdEncoding.EncodeToString(nonceBytes)
			body = []byte(strings.ReplaceAll(string(body), "<script>", `<script nonce="`+nonce+`">`))
			w.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'nonce-"+nonce+"'")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(body)
			return
		}
		clone := r.Clone(r.Context())
		clone.URL.Path = "/" + clean
		files.ServeHTTP(w, clone)
	})
}
