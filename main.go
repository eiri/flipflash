package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	dir := flag.String("dir", "", "directory of images (required)")
	port := flag.Int("port", 8081, "HTTP port")
	flag.Parse()

	if *dir == "" || *port < 1 || *port > 65535 || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: flipflash --dir DIRECTORY [--port 8081]")
		os.Exit(2)
	}

	root, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil {
		log.Fatal(err)
	}
	if !info.IsDir() {
		log.Fatal(errors.New("--dir must be a directory"))
	}

	gallery, err := newGallery(root)
	if err != nil {
		log.Fatal(err)
	}

	addr := "0.0.0.0:" + strconv.Itoa(*port)
	log.Printf("Serving %d images from %s at http://%s", len(gallery.images), root, addr)
	server := &http.Server{
		Addr:              addr,
		Handler:           gallery.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
