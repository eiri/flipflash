# FlipFlash

FlipFlash is a simple web gallery server. I made it to check out all tryout images I build with mflux. FlipFlash supports JPEG, PNG, GIF, WebP, and AVIF files.

## Build and run

Install [mise](https://mise.jdx.dev/), then run:

```sh
mise install
mise run build
./flipflash --dir /path/to/photos
```

Open `http://localhost:8081` in a browser. Add `--port 9000` to use a different port. The server listens on `0.0.0.0`, so other devices on your network can connect, so do not run it on an untrusted network, coz anyone who can reach the server can view the images.

If you do not use mise, install Go 1.26 and run `go build -o flipflash .`.

## Using the gallery

The grid shows thumbnails. The scroll icon make images list that fits each image on the screen. Click on an image to open it in a view. Click the top of the image to return to the gallery, or click its left or right edge to move between images. The arrow keys move between images, and Escape closes the viewer.

Files appear in order by relative path. FlipFlash skips symbolic links. It makes JPEG, PNG, and GIF thumbnails when the browser asks for them. For WebP and AVIF, it sends the original files. The server does not store thumbnails, but browsers can cache generated thumbnails for one hour.

## Development

Run `mise run check` to make sure that Go files are formatted, run tests and `go vet`, and build the binary. GitHub Actions runs the same task on pushes to `main` and on pull requests. Dependabot looks for updates to Go modules and GitHub Actions each week.

## License

[MIT](LICENSE)
