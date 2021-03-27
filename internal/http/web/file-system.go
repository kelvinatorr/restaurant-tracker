package web

import (
	"log"
	"net/http"
	"strings"
)

type fileSystem struct {
	fs http.FileSystem
}

// Open opens file if it exists in the directory. If path is a directory it checks if index.html exists
// and returns nil, err if it does not.
func (fs fileSystem) Open(path string) (http.File, error) {
	log.Printf("fs Opening Path: %s", path)
	f, err := fs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		// Check if index.html exists. If it does not return 404.
		// Without this, the default behavior is to return a directory listing, which we don't want.
		index := strings.TrimSuffix(path, "/") + "/index.html"
		if _, err := fs.fs.Open(index); err != nil {
			return nil, err
		}
	}
	return f, nil
}
