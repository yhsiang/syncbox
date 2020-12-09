package syncbox

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
)

//go:generate callbackgen -type FileWatcher
type FileWatcher struct {
	mu    sync.Mutex
	path  string
	ctx   context.Context
	files map[string]File

	changeCallbacks []func(files []File)
}

type File struct {
	Name     string    `json:"name"`
	RootPath string    `json:"-"`
	Path     string    `json:"path"`
	Checksum string    `json:"checksum"`
	State    string    `json:"-"`
	Action   string    `json:"action"`
	Content  io.Reader `json:"-"`
	// isDirectory bool
	// Size        int64
}

func (f *File) CalChecksum() error {
	file, err := os.Open(strings.Join([]string{f.RootPath, f.Path}, ""))
	if err != nil {
		return err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	f.Checksum = hex.EncodeToString(hash.Sum(nil))

	return nil
}

func (f *FileWatcher) WalkDir() error {
	var changes []File
	var newFiles = make(map[string]File)
	err := filepath.Walk(f.path, func(path string, info os.FileInfo, err error) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		if err != nil {
			log.WithError(err).Error("walk error")
		}

		if !info.IsDir() {
			var cuttedPath = strings.Replace(path, f.path, "", 1)
			file := File{
				Name:     info.Name(),
				RootPath: f.path,
				Path:     cuttedPath,
			}

			err2 := file.CalChecksum()
			if err2 != nil {
				return err2
			}

			log.Debugf("%+v", file)
			newFiles[file.Path] = file
		}

		return nil
	})

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, oldFile := range f.files {
		newFile, ok := newFiles[oldFile.Path]
		if !ok {
			oldFile.State = "delete"
			changes = append(changes, oldFile)
		}

		if ok && oldFile.Checksum != newFile.Checksum {
			oldFile.State = "update"
			changes = append(changes, oldFile)
		}
	}

	for _, newFile := range newFiles {
		if _, ok := f.files[newFile.Path]; !ok {
			newFile.State = "new"
			changes = append(changes, newFile)
		}
	}

	f.files = newFiles

	if len(changes) > 0 {
		f.EmitChange(changes)
	}

	return err
}

func (f *FileWatcher) Run() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	err := f.WalkDir()
	if err != nil {
		log.WithError(err).Error("run error")
	}

	for {
		select {
		case <-f.ctx.Done():
			return
		case <-ticker.C:
			err := f.WalkDir()
			if err != nil {
				log.WithError(err).Error("run error")
			}
		}
	}
}

func (f *FileWatcher) Compare(files []File) []File {
	f.mu.Lock()
	defer f.mu.Unlock()

	var syncingFiles []File
	for _, file := range files {
		if _, ok := f.files[file.Path]; !ok {
			file.Action = "upload"
			syncingFiles = append(syncingFiles, file)
		}
	}
	// TODO: handle downloading
	return syncingFiles
}

func NewFileWatcher(ctx context.Context, path string) *FileWatcher {
	return &FileWatcher{
		path:  path,
		ctx:   ctx,
		files: make(map[string]File),
	}
}
