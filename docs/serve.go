package main

import (
	"embed"
	"net/http"
)

//go:embed index.html worker.wasm wasm_exec.js
var fs embed.FS

func main() {
	http.Handle("/", http.FileServer(http.FS(fs)))
	http.ListenAndServe(":2288", nil)
}
