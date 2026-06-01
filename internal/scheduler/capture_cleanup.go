package scheduler

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"lumenroute/internal/capture"
	"lumenroute/internal/project"
)

// StartCaptureCleanup runs hourly to delete capture records and JSONL files
// older than each project's retention_days setting.
func StartCaptureCleanup(projectSvc *project.Service, captureStore *capture.Store, basePath string, quit <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runCaptureCleanup(projectSvc, captureStore, basePath)
			case <-quit:
				return
			}
		}
	}()
}

func runCaptureCleanup(projectSvc *project.Service, captureStore *capture.Store, basePath string) {
	projects, err := projectSvc.ListWithRetention()
	if err != nil {
		log.Printf("capture cleanup: list projects: %v", err)
		return
	}

	for _, p := range projects {
		cutoff := time.Now().Add(-time.Duration(p.RetentionDays) * 24 * time.Hour)
		filePaths, deleted, err := captureStore.DeleteByProjectBefore(p.ID, cutoff)
		if err != nil {
			log.Printf("capture cleanup project %d: %v", p.ID, err)
			continue
		}
		if deleted > 0 {
			log.Printf("capture cleanup: project %d (%s): removed %d records", p.ID, p.Name, deleted)
		}

		for _, fp := range filePaths {
			fullPath := filepath.Join(basePath, fp)
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}
			if time.Since(info.ModTime()) < 1*time.Hour {
				continue
			}
			remaining, _ := captureStore.CountByProject(p.ID)
			if remaining == 0 {
				if err := os.Remove(fullPath); err == nil {
					log.Printf("capture cleanup: removed file %s", fp)
				}
			}
		}
	}
}
