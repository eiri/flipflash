package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGallery(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatal(err)
	}
	img := image.NewRGBA(image.Rect(0, 0, 720, 480))
	img.Set(0, 0, color.White)
	file, err := os.Create(filepath.Join(root, "nested", "picture.PNG"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("not an image"), 0644); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Symlink(filepath.Join(root, "nested", "picture.PNG"), filepath.Join(root, "linked.png")); err != nil {
			t.Fatal(err)
		}
	}

	g, err := newGallery(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.images) != 1 || g.images[0].Name != "nested/picture.PNG" {
		t.Fatalf("images = %+v", g.images)
	}

	server := httptest.NewServer(g.routes())
	defer server.Close()
	resp, err := http.Get(server.URL + "/api/images")
	if err != nil {
		t.Fatal(err)
	}
	var images []photo
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if len(images) != 1 || images[0].Name != "nested/picture.PNG" {
		t.Fatalf("API images = %+v", images)
	}

	for _, path := range []string{"/", "/images/0", "/thumbs/0"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status = %d", path, resp.StatusCode)
		}
		if path == "/thumbs/0" {
			thumb, _, err := image.Decode(resp.Body)
			if err != nil || thumb.Bounds().Dx() != 360 || thumb.Bounds().Dy() != 240 {
				t.Errorf("thumbnail = %v, %v", thumb, err)
			}
		}
		resp.Body.Close()
	}
	for _, path := range []string{"/images/1", "/thumbs/-1", "/images/not-an-id", "/other"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d", path, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestEmptyGallery(t *testing.T) {
	g, err := newGallery(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	g.routes().ServeHTTP(recorder, httptest.NewRequest("GET", "/api/images", nil))
	if recorder.Body.String() != "[]\n" {
		t.Fatalf("empty images = %q", recorder.Body.String())
	}
}
