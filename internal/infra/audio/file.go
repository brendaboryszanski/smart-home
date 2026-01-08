package audio

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileSource struct {
	dir       string
	processed map[string]bool
	mu        sync.Mutex
}

func NewFileSource(dir string) *FileSource {
	return &FileSource{
		dir:       dir,
		processed: make(map[string]bool),
	}
}

func (f *FileSource) Name() string {
	return "file"
}

func (f *FileSource) Start(_ context.Context) error {
	if err := os.MkdirAll(f.dir, 0755); err != nil {
		return fmt.Errorf("creating audio dir: %w", err)
	}
	return nil
}

func (f *FileSource) Stop() error {
	return nil
}

func (f *FileSource) NextCommand(ctx context.Context) ([]byte, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			audio, err := f.checkForNewFile()
			if err != nil {
				return nil, err
			}
			if audio != nil {
				return audio, nil
			}
		}
	}
}

func (f *FileSource) checkForNewFile() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entries, err := os.ReadDir(f.dir)
	if err != nil {
		return nil, fmt.Errorf("reading dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".wav" && ext != ".mp3" && ext != ".m4a" && ext != ".webm" {
			continue
		}

		path := filepath.Join(f.dir, entry.Name())
		if f.processed[path] {
			continue
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", path, err)
		}

		f.processed[path] = true

		processedPath := path + ".processed"
		os.Rename(path, processedPath)

		return data, nil
	}

	return nil, nil
}

