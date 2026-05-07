package scheduler

import (
	"log"
	"net/http"
	"time"

	"lumenroute/internal/metrics"
	"lumenroute/internal/provider"
)

func StartHealthChecker(ps *provider.Service, intervalSeconds int, quit <-chan struct{}, rec metrics.Recorder) {
	if intervalSeconds <= 0 {
		intervalSeconds = 30
	}
	client := &http.Client{Timeout: 5 * time.Second}
	go func() {
		ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				providers, err := ps.List()
				if err != nil {
					log.Printf("health checker list providers: %v", err)
					continue
				}
				var healthy, unhealthy int64
				for _, p := range providers {
					if !p.Enabled {
						continue
					}
					status, code, latency, lastErr := checkProvider(client, p.BaseURL, p.HealthCheckPath)
					ps.UpdateHealth(p.ID, status, code, latency, lastErr)
					if status == "healthy" {
						healthy++
					} else {
						unhealthy++
					}
				}
				if rec != nil {
					rec.SetProviderHealthCounts(healthy, unhealthy)
				}
			case <-quit:
				return
			}
		}
	}()
}

func checkProvider(client *http.Client, baseURL, healthPath string) (status string, code int, latency int, lastErr string) {
	if healthPath == "" {
		healthPath = "/models"
	}
	url := baseURL + healthPath
	start := time.Now()
	resp, err := client.Get(url)
	latency = int(time.Since(start).Milliseconds())
	if err != nil {
		return "unhealthy", 0, latency, err.Error()
	}
	defer resp.Body.Close()
	code = resp.StatusCode
	if code >= 200 && code < 300 {
		return "healthy", code, latency, ""
	}
	return "unhealthy", code, latency, "unexpected status code"
}

