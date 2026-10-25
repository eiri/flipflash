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
	if err := os.WriteFile(filepath.Join(root, "sample.webp"), []byte("RIFF0000WEBP"), 0644); err != nil {
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
	t.Cleanup(func() { g.root.Close() })
	if len(g.images) != 2 || g.images[0].Name != "nested/picture.PNG" || g.images[1].Name != "sample.webp" {
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
	if len(images) != 2 || images[0].Name != "nested/picture.PNG" || images[1].Name != "sample.webp" {
		t.Fatalf("API images = %+v", images)
	}

	for _, path := range []string{"/", "/images/0", "/thumbs/0", "/images/1", "/thumbs/1"} {
		resp, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: status = %d", path, resp.StatusCode)
		}
		if path != "/" && resp.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("%s: Cache-Control = %q", path, resp.Header.Get("Cache-Control"))
		}
		if path == "/thumbs/0" {
			thumb, _, err := image.Decode(resp.Body)
			if err != nil || thumb.Bounds().Dx() != 360 || thumb.Bounds().Dy() != 240 {
				t.Errorf("thumbnail = %v, %v", thumb, err)
			}
		}
		resp.Body.Close()
	}
	for _, path := range []string{"/images/2", "/thumbs/-1", "/images/not-an-id", "/other"} {
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

func TestSymlinkSwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require additional privileges")
	}

	for _, ext := range []string{".jpg", ".png", ".gif", ".webp", ".avif"} {
		for _, parent := range []bool{false, true} {
			name := ext + "/file"
			if parent {
				name = ext + "/directory"
			}
			t.Run(name, func(t *testing.T) {
				root, outside := t.TempDir(), t.TempDir()
				nested := filepath.Join(root, "nested")
				if err := os.Mkdir(nested, 0755); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(nested, "photo"+ext)
				if err := os.WriteFile(path, []byte("original"), 0644); err != nil {
					t.Fatal(err)
				}
				g, err := newGallery(root)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { g.root.Close() })

				// Replace an indexed path after scanning; neither route may escape.
				secret := filepath.Join(outside, "photo"+ext)
				if err := os.WriteFile(secret, []byte("outside secret"), 0644); err != nil {
					t.Fatal(err)
				}
				if parent {
					path, secret = nested, outside
				}
				if err := os.RemoveAll(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(secret, path); err != nil {
					t.Fatal(err)
				}
				for _, route := range []string{"/images/0", "/thumbs/0"} {
					w := httptest.NewRecorder()
					g.routes().ServeHTTP(w, httptest.NewRequest("GET", route, nil))
					if w.Code < 400 || w.Body.String() == "outside secret" {
						t.Errorf("%s: status = %d, body = %q", route, w.Code, w.Body.String())
					}
				}
			})
		}
	}
}

func TestEmptyGallery(t *testing.T) {
	g, err := newGallery(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { g.root.Close() })
	recorder := httptest.NewRecorder()
	g.routes().ServeHTTP(recorder, httptest.NewRequest("GET", "/api/images", nil))
	if recorder.Body.String() != "[]\n" {
		t.Fatalf("empty images = %q", recorder.Body.String())
	}
}
