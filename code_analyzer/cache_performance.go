package code_analyzer

import (
	"time"
)

// recordCacheHit increments cache hit counter
func (cm *CacheManager) recordCacheHit() {
	if cm.stats == nil {
		return
	}
	cm.stats.mutex.Lock()
	defer cm.stats.mutex.Unlock()
	cm.stats.TotalRequests++
	cm.stats.CacheHits++
}

// recordCacheMiss increments cache miss counter
func (cm *CacheManager) recordCacheMiss() {
	if cm.stats == nil {
		return
	}
	cm.stats.mutex.Lock()
	defer cm.stats.mutex.Unlock()
	cm.stats.TotalRequests++
	cm.stats.CacheMisses++
}

// GetPerformanceStats returns detailed cache performance statistics
func (cm *CacheManager) GetPerformanceStats() map[string]interface{} {
	if cm.stats == nil {
		return map[string]interface{}{
			"total_requests":      0,
			"cache_hits":          0,
			"cache_misses":        0,
			"hit_rate_percent":    0.0,
			"miss_rate_percent":   0.0,
			"uptime_seconds":      0.0,
			"uptime_human":        "0s",
			"requests_per_second": 0.0,
		}
	}

	cm.stats.mutex.RLock()
	defer cm.stats.mutex.RUnlock()

	hitRate := 0.0
	if cm.stats.TotalRequests > 0 {
		hitRate = float64(cm.stats.CacheHits) / float64(cm.stats.TotalRequests) * 100
	}

	missRate := 0.0
	if cm.stats.TotalRequests > 0 {
		missRate = float64(cm.stats.CacheMisses) / float64(cm.stats.TotalRequests) * 100
	}

	uptime := time.Since(cm.stats.LastResetTime)

	reqPerSec := 0.0
	if uptime.Seconds() > 0 {
		reqPerSec = float64(cm.stats.TotalRequests) / uptime.Seconds()
	}

	return map[string]interface{}{
		"total_requests":      cm.stats.TotalRequests,
		"cache_hits":          cm.stats.CacheHits,
		"cache_misses":        cm.stats.CacheMisses,
		"hit_rate_percent":    hitRate,
		"miss_rate_percent":   missRate,
		"uptime_seconds":      uptime.Seconds(),
		"uptime_human":        uptime.String(),
		"requests_per_second": reqPerSec,
		"last_reset":          cm.stats.LastResetTime.Format(time.RFC3339),
	}
}

// ResetPerformanceStats resets all performance counters
func (cm *CacheManager) ResetPerformanceStats() {
	if cm.stats == nil {
		return
	}
	cm.stats.mutex.Lock()
	defer cm.stats.mutex.Unlock()

	cm.stats.TotalRequests = 0
	cm.stats.CacheHits = 0
	cm.stats.CacheMisses = 0
	cm.stats.LastResetTime = time.Now()
}
