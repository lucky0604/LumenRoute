package capture

import (
	"container/list"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileWriter handles JSONL file writes with an LRU file handle pool.
type FileWriter struct {
	basePath string
	pool     *lruFilePool
}

func NewFileWriter(basePath string, poolSize int) *FileWriter {
	return &FileWriter{
		basePath: basePath,
		pool:     newLRUFilePool(poolSize),
	}
}

// Write appends a capture entry as a JSONL line and returns the relative
// file path and byte offset of the written line.
func (w *FileWriter) Write(entry CaptureEntry) (string, int64, error) {
	relPath := w.buildRelPath(entry.ProjectID)
	absPath := filepath.Join(w.basePath, relPath)

	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", 0, fmt.Errorf("mkdir %s: %w", dir, err)
	}

	f, err := w.pool.Get(absPath)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", absPath, err)
	}

	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return "", 0, fmt.Errorf("seek %s: %w", absPath, err)
	}

	line := marshalJSONL(entry)
	line = append(line, '\n')

	n, err := f.Write(line)
	if err != nil {
		return "", 0, fmt.Errorf("write %s: %w", absPath, err)
	}
	if n != len(line) {
		return "", 0, fmt.Errorf("short write %s: wrote %d of %d", absPath, n, len(line))
	}

	return relPath, offset, nil
}

func (w *FileWriter) Sync() {
	w.pool.SyncAll()
}

func (w *FileWriter) Close() {
	w.pool.CloseAll()
}

func (w *FileWriter) buildRelPath(projectID int64) string {
	now := time.Now()
	return filepath.Join(
		fmt.Sprintf("%d", projectID),
		now.Format("2006-01-02"),
		now.Format("15")+".jsonl",
	)
}

func marshalJSONL(entry CaptureEntry) []byte {
	line := jsonlLine{
		RequestID:         entry.RequestID,
		ProjectID:         entry.ProjectID,
		RouteName:         entry.RouteName,
		PublicModelName:   entry.PublicModelName,
		UpstreamModelName: entry.UpstreamModelName,
		ProviderName:      entry.ProviderName,
		Stream:            entry.Stream,
		StatusCode:        entry.StatusCode,
		LatencyMs:         entry.LatencyMs,
		TTFCMs:            entry.TTFCMs,
		PromptTokens:      entry.PromptTokens,
		CompletionTokens:  entry.CompletionTokens,
		RequestBody:       entry.RequestBody,
		ResponseBody:      entry.ResponseBody,
		CapturedAt:        time.Now().UTC(),
		SchemaVersion:     1,
	}
	if entry.Stream {
		line.StreamCompleted = &entry.StreamCompleted
	}
	data, _ := json.Marshal(line)
	return data
}

// lruFilePool manages a fixed-size pool of open file handles, evicting
// the least recently used when the pool is full.
type lruFilePool struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

type poolEntry struct {
	path string
	file *os.File
}

func newLRUFilePool(capacity int) *lruFilePool {
	if capacity <= 0 {
		capacity = 16
	}
	return &lruFilePool{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (p *lruFilePool) Get(path string) (*os.File, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.items[path]; ok {
		p.order.MoveToFront(elem)
		return elem.Value.(*poolEntry).file, nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	if p.order.Len() >= p.capacity {
		back := p.order.Back()
		if back != nil {
			evicted := p.order.Remove(back).(*poolEntry)
			evicted.file.Sync()
			evicted.file.Close()
			delete(p.items, evicted.path)
		}
	}

	entry := &poolEntry{path: path, file: f}
	elem := p.order.PushFront(entry)
	p.items[path] = elem
	return f, nil
}

func (p *lruFilePool) SyncAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for e := p.order.Front(); e != nil; e = e.Next() {
		e.Value.(*poolEntry).file.Sync()
	}
}

func (p *lruFilePool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for e := p.order.Front(); e != nil; e = e.Next() {
		pe := e.Value.(*poolEntry)
		pe.file.Sync()
		pe.file.Close()
	}
	p.items = make(map[string]*list.Element)
	p.order.Init()
}
