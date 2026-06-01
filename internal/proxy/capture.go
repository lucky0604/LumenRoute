package proxy

import (
	"math/rand"
	"sync"
	"time"

	"lumenroute/internal/capture"
	"lumenroute/internal/project"
)

type projectCacheEntry struct {
	project   *project.Project
	expiresAt time.Time
}

type projCache struct {
	mu    sync.RWMutex
	items map[int64]*projectCacheEntry
	ttl   time.Duration
}

func newProjCache(ttl time.Duration) *projCache {
	return &projCache{
		items: make(map[int64]*projectCacheEntry),
		ttl:   ttl,
	}
}

func (c *projCache) get(id int64) (*project.Project, bool) {
	c.mu.RLock()
	entry, ok := c.items[id]
	c.mu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.project, true
}

func (c *projCache) set(id int64, p *project.Project) {
	c.mu.Lock()
	c.items[id] = &projectCacheEntry{
		project:   p,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *projCache) invalidate(id int64) {
	c.mu.Lock()
	delete(c.items, id)
	c.mu.Unlock()
}

// maybeCapture checks capture eligibility and submits the entry.
func (s *Service) maybeCapture(entry capture.CaptureEntry) {
	if s.captureService == nil || !s.captureEnabled {
		return
	}
	if entry.ProjectID == 0 {
		return
	}

	proj := s.getProjectCached(entry.ProjectID)
	if proj == nil || !shouldCapture(proj) {
		return
	}

	maxSize := s.captureMaxBodySize
	if maxSize > 0 && (len(entry.RequestBody) > maxSize || len(entry.ResponseBody) > maxSize) {
		entry.RequestBody = nil
		entry.ResponseBody = nil
		entry.BodySkipped = true
	}

	s.captureService.Submit(entry)
}

func (s *Service) getProjectCached(projectID int64) *project.Project {
	if p, ok := s.cache.get(projectID); ok {
		return p
	}

	p, err := s.projectService.Get(projectID)
	if err != nil {
		return nil
	}

	s.cache.set(projectID, p)
	return p
}

func shouldCapture(p *project.Project) bool {
	if !p.CaptureEnabled {
		return false
	}
	if p.SampleRate >= 1.0 {
		return true
	}
	if p.SampleRate <= 0 {
		return false
	}
	return rand.Float64() < p.SampleRate
}
