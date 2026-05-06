package scheduler

import (
	"database/sql"
	"log"
	"time"
)

func StartLogCleanup(db *sql.DB, retentionDays int, quit <-chan struct{}) {
	if retentionDays <= 0 {
		retentionDays = 7
	}
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
				result, err := db.Exec(`DELETE FROM request_logs WHERE created_at < ?`, cutoff)
				if err != nil {
					log.Printf("log cleanup: %v", err)
					continue
				}
				n, _ := result.RowsAffected()
				if n > 0 {
					log.Printf("log cleanup: removed %d old records", n)
				}
			case <-quit:
				return
			}
		}
	}()
}
