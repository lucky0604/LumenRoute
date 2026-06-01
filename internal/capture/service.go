package capture

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// Service manages the asynchronous capture pipeline.
type Service struct {
	entries chan CaptureEntry
	writer  *FileWriter
	store   *Store
	dropped atomic.Int64
	wg      sync.WaitGroup
	config  Config
}

func NewService(cfg Config, store *Store) *Service {
	channelSize := cfg.ChannelSize
	if channelSize <= 0 {
		channelSize = 256
	}
	return &Service{
		entries: make(chan CaptureEntry, channelSize),
		writer:  NewFileWriter(cfg.BasePath, 16),
		store:   store,
		config:  cfg,
	}
}

// Submit enqueues a capture entry. Non-blocking: drops if channel is full.
func (s *Service) Submit(entry CaptureEntry) {
	select {
	case s.entries <- entry:
		captureTotal.Add(1)
	default:
		s.dropped.Add(1)
		captureDroppedTotal.Add(1)
	}
}

// Start begins the background writer goroutine.
func (s *Service) Start() {
	s.wg.Add(1)
	go s.processLoop()
}

// Close signals the writer to drain remaining entries and shut down.
func (s *Service) Close() {
	close(s.entries)
	s.wg.Wait()
	s.writer.Close()
}

// Dropped returns the total number of entries dropped due to full channel.
func (s *Service) Dropped() int64 {
	return s.dropped.Load()
}

// QueueLength returns the current number of entries waiting in the channel.
func (s *Service) QueueLength() int {
	return len(s.entries)
}

func (s *Service) processLoop() {
	defer s.wg.Done()

	batchSize := s.config.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	batch := make([]CaptureRecord, 0, batchSize)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	const maxRetries = 3
	flush := func() {
		if len(batch) == 0 {
			return
		}
		s.writer.Sync()
		var err error
		for attempt := 0; attempt < maxRetries; attempt++ {
			if err = s.store.InsertBatch(batch); err == nil {
				break
			}
			log.Printf("[capture] store batch retry %d/%d (%d records): %v", attempt+1, maxRetries, len(batch), err)
			time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
		}
		if err != nil {
			captureStoreErrorsTotal.Add(int64(len(batch)))
			log.Printf("[capture] store batch failed after %d retries, dropping %d records", maxRetries, len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case entry, ok := <-s.entries:
			if !ok {
				flush()
				return
			}

			if entry.BodySkipped {
				batch = append(batch, CaptureRecord{
					RequestID:       entry.RequestID,
					ProjectID:       entry.ProjectID,
					PublicModelName: entry.PublicModelName,
					Stream:          entry.Stream,
					StatusCode:      entry.StatusCode,
					BodySkipped:     true,
					FilePath:        "",
					FileOffset:      0,
					RequestSize:     0,
					ResponseSize:    0,
				})
			} else {
				filePath, offset, err := s.writer.Write(entry)
				if err != nil {
					captureWriteErrorsTotal.Add(1)
					log.Printf("[capture] write error: %v", err)
					continue
				}
				batch = append(batch, CaptureRecord{
					RequestID:       entry.RequestID,
					ProjectID:       entry.ProjectID,
					PublicModelName: entry.PublicModelName,
					Stream:          entry.Stream,
					StatusCode:      entry.StatusCode,
					BodySkipped:     false,
					FilePath:        filePath,
					FileOffset:      offset,
					RequestSize:     len(entry.RequestBody),
					ResponseSize:    len(entry.ResponseBody),
				})
			}

			if len(batch) >= batchSize {
				flush()
			}

		case <-ticker.C:
			flush()
		}
	}
}
