package staticembed

import (
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-contrib/static"
)

type sfs struct {
	http.FileSystem
}

func (s *sfs) Exists(prefix string, path string) bool {
	p := filepath.Join(prefix, path)
	log.Print(p)
	_, err := s.Open(p)
	return err != nil
}

func FS(fs fs.FS) static.ServeFileSystem {
	return &sfs{http.FS(fs)}
}
