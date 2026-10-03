package main

import (
	"embed"
	"encoding/json"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

//go:embed web/index.html
var pages embed.FS

type photo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	path string
}

type gallery struct {
	images  []photo
	workers chan struct{}
}

var extensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".avif": true,
}

func newGallery(root string) (*gallery, error) {
	images := make([]photo, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Do not follow links outside the selected directory.
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() || !extensions[strings.ToLower(filepath.Ext(entry.Name()))] {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		images = append(images, photo{Name: filepath.ToSlash(rel), path: path})
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(images, func(a, b photo) int { return strings.Compare(a.Name, b.Name) })
	for i := range images {
		images[i].ID = i
	}
	return &gallery{images: images, workers: make(chan struct{}, 4)}, nil
}

func (g *gallery) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, pages, "web/index.html")
	})
	mux.HandleFunc("GET /api/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(g.images)
	})
	mux.HandleFunc("GET /images/{id}", g.serveImage)
	mux.HandleFunc("GET /thumbs/{id}", g.serveThumb)
	return mux
}

func (g *gallery) image(r *http.Request) (photo, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 0 || id >= len(g.images) {
		return photo{}, false
	}
	return g.images[id], true
}

func (g *gallery) serveImage(w http.ResponseWriter, r *http.Request) {
	photo, ok := g.image(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, photo.path)
}

func (g *gallery) serveThumb(w http.ResponseWriter, r *http.Request) {
	photo, ok := g.image(r)
	if !ok {
		http.NotFound(w, r)
		return
	}

	// Formats without a standard-library decoder use the original file.
	ext := strings.ToLower(filepath.Ext(photo.path))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" {
		http.ServeFile(w, r, photo.path)
		return
	}

	select {
	case g.workers <- struct{}{}:
		defer func() { <-g.workers }()
	case <-r.Context().Done():
		return
	}

	file, err := os.Open(photo.path)
	if err != nil {
		http.Error(w, "Could not read image", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil || config.Width <= 0 || config.Height <= 0 {
		http.Error(w, "Invalid image", http.StatusUnsupportedMediaType)
		return
	}
	// Avoid decoding excessively large images into memory.
	if int64(config.Width)*int64(config.Height) > 25_000_000 {
		http.ServeFile(w, r, photo.path)
		return
	}
	if _, err = file.Seek(0, 0); err != nil {
		http.Error(w, "Could not read image", http.StatusInternalServerError)
		return
	}
	img, _, err := image.Decode(file)
	if err != nil {
		http.Error(w, "Invalid image", http.StatusUnsupportedMediaType)
		return
	}

	thumb := shrink(img, 360)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_ = jpeg.Encode(w, thumb, &jpeg.Options{Quality: 78})
}

func shrink(src image.Image, limit int) image.Image {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= limit && height <= limit {
		return src
	}
	scale := float64(limit) / float64(width)
	if height > width {
		scale = float64(limit) / float64(height)
	}
	w, h := max(1, int(float64(width)*scale)), max(1, int(float64(height)*scale))
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			out.Set(x, y, src.At(bounds.Min.X+x*width/w, bounds.Min.Y+y*height/h))
		}
	}
	return out
}
