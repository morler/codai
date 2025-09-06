package code_analyzer

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/meysamhadeli/codai/code_analyzer/models"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Data      interface{}
	Timestamp time.Time
	FileSize  int64
	ModTime   time.Time
	Hash      string
}

// FileCache manages file-based caching with intelligent invalidation
type FileCache struct {
	cacheDir string
	mutex    sync.RWMutex
}

// CacheStats tracks cache performance metrics
type CacheStats struct {
	TotalRequests  int64
	CacheHits      int64
	CacheMisses    int64
	TotalSizeBytes int64
	LastResetTime  time.Time
	mutex          sync.RWMutex
}

// CacheManager provides high-level caching operations
type CacheManager struct {
	fileCache *FileCache
	stats     *CacheStats
}

// NewCacheManager creates a new cache manager instance
// If cacheDir is empty, it defaults to "cache" directory in the current working directory
func NewCacheManager(cacheDir string) (*CacheManager, error) {
	// Register types for gob encoding/decoding
	gob.Register(&models.FullContextData{})
	gob.Register([]models.FileData{})
	gob.Register([]string{})
	gob.Register([]byte{})
	gob.Register(&models.ProjectSnapshot{})
	gob.Register(models.FileSnapshot{})

	if cacheDir == "" {
		// Get current working directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}
		cacheDir = filepath.Join(cwd, ".cache")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	fileCache := &FileCache{
		cacheDir: cacheDir,
	}

	cacheManager := &CacheManager{
		fileCache: fileCache,
		stats: &CacheStats{
			LastResetTime: time.Now(),
		},
	}

	// Perform automatic cleanup on initialization (background cleanup)
	go cacheManager.performAutoCleanup()

	return cacheManager, nil
}

// generateCacheKey creates a unique cache key for a file
func (fc *FileCache) generateCacheKey(filePath string) string {
	hash := md5.Sum([]byte(filePath))
	return fmt.Sprintf("%x.cache", hash)
}

// getCachePath returns the full path to a cache file
func (fc *FileCache) getCachePath(cacheKey string) string {
	return filepath.Join(fc.cacheDir, cacheKey)
}

// isFileChanged checks if a file has been modified since last cache
func (fc *FileCache) isFileChanged(filePath string, entry *CacheEntry) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return true, err
	}

	// Check modification time and file size
	if !fileInfo.ModTime().Equal(entry.ModTime) || fileInfo.Size() != entry.FileSize {
		return true, nil
	}

	return false, nil
}

// Get retrieves data from cache if valid, returns nil if not found or invalid
func (fc *FileCache) Get(filePath string) (interface{}, bool) {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()

	cacheKey := fc.generateCacheKey(filePath)
	cachePath := fc.getCachePath(cacheKey)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, false
	}

	// Read cache file
	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	var entry CacheEntry
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&entry); err != nil {
		return nil, false
	}

	// Check if original file has changed
	changed, err := fc.isFileChanged(filePath, &entry)
	if err != nil || changed {
		// File has changed or error occurred, invalidate cache
		os.Remove(cachePath)
		return nil, false
	}

	return entry.Data, true
}

// Set stores data in cache with file metadata
func (fc *FileCache) Set(filePath string, data interface{}) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	entry := CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		FileSize:  fileInfo.Size(),
		ModTime:   fileInfo.ModTime(),
		Hash:      fc.generateCacheKey(filePath),
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode cache entry: %w", err)
	}
	gobData := buffer.Bytes()

	cacheKey := fc.generateCacheKey(filePath)
	cachePath := fc.getCachePath(cacheKey)

	if err := ioutil.WriteFile(cachePath, gobData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Delete removes a cache entry
func (fc *FileCache) Delete(filePath string) error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	cacheKey := fc.generateCacheKey(filePath)
	cachePath := fc.getCachePath(cacheKey)

	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	return nil
}

// Clear removes all cache files
func (fc *FileCache) Clear() error {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	return os.RemoveAll(fc.cacheDir)
}

// GetConfigCache retrieves cached configuration data
func (cm *CacheManager) GetConfigCache(configPath string) (*models.FullContextData, bool) {
	data, found := cm.fileCache.Get(configPath)
	if !found {
		cm.recordCacheMiss()
		return nil, false
	}

	// Type assertion to convert back to FullContextData
	if contextData, ok := data.(*models.FullContextData); ok {
		cm.recordCacheHit()
		return contextData, true
	}

	cm.recordCacheMiss()
	return nil, false
}

// SetConfigCache stores configuration data in cache
func (cm *CacheManager) SetConfigCache(configPath string, data *models.FullContextData) error {
	return cm.fileCache.Set(configPath, data)
}

// GetFileContentCache retrieves cached file content
func (cm *CacheManager) GetFileContentCache(filePath string) ([]byte, bool) {
	data, found := cm.fileCache.Get(filePath)
	if !found {
		cm.recordCacheMiss()
		return nil, false
	}

	if content, ok := data.([]byte); ok {
		cm.recordCacheHit()
		return content, true
	}

	cm.recordCacheMiss()
	return nil, false
}

// SetFileContentCache stores file content in cache
func (cm *CacheManager) SetFileContentCache(filePath string, content []byte) error {
	return cm.fileCache.Set(filePath, content)
}

// GetTreeSitterCache retrieves cached tree-sitter parsing results
func (cm *CacheManager) GetTreeSitterCache(filePath string) ([]string, bool) {
	cm.fileCache.mutex.RLock()
	defer cm.fileCache.mutex.RUnlock()

	cacheKey := cm.fileCache.generateCacheKey(filePath + ".treesitter")
	cachePath := cm.fileCache.getCachePath(cacheKey)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		cm.recordCacheMiss()
		return nil, false
	}

	// Read cache file
	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		cm.recordCacheMiss()
		return nil, false
	}

	// Decode the cache entry
	var entry CacheEntry
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&entry); err != nil {
		cm.recordCacheMiss()
		return nil, false
	}

	// Extract tree-sitter results from entry
	if codeParts, ok := entry.Data.([]string); ok {
		cm.recordCacheHit()
		return codeParts, true
	}

	cm.recordCacheMiss()
	return nil, false
}

// SetTreeSitterCache stores tree-sitter parsing results in cache
func (cm *CacheManager) SetTreeSitterCache(filePath string, codeParts []string) error {
	cm.fileCache.mutex.Lock()
	defer cm.fileCache.mutex.Unlock()

	entry := CacheEntry{
		Data:      codeParts,
		Timestamp: time.Now(),
		FileSize:  0, // Not applicable for tree-sitter results
		ModTime:   time.Now(),
		Hash:      filePath + ".treesitter",
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode tree-sitter entry: %w", err)
	}

	cacheKey := cm.fileCache.generateCacheKey(filePath + ".treesitter")
	cachePath := cm.fileCache.getCachePath(cacheKey)

	if err := ioutil.WriteFile(cachePath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write tree-sitter cache file: %w", err)
	}

	return nil
}

// GetCacheStats returns cache statistics
func (cm *CacheManager) GetCacheStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count cache files
	files, err := ioutil.ReadDir(cm.fileCache.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var totalSize int64
	for _, file := range files {
		if !file.IsDir() {
			totalSize += file.Size()
		}
	}

	stats["cache_files"] = len(files)
	stats["total_size"] = totalSize
	stats["cache_dir"] = cm.fileCache.cacheDir

	return stats, nil
}

// GetDetailedCacheStats returns detailed cache statistics including file counts by type
func (cm *CacheManager) GetDetailedCacheStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	files, err := ioutil.ReadDir(cm.fileCache.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var totalSize int64
	var fileContentCount, treeSitterCount, snapshotCount, configCount int
	oldestTime := time.Now()
	newestTime := time.Time{}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		totalSize += file.Size()
		modTime := file.ModTime()

		if modTime.Before(oldestTime) {
			oldestTime = modTime
		}
		if modTime.After(newestTime) {
			newestTime = modTime
		}

		// Analyze cache entry type by reading the data
		cachePath := filepath.Join(cm.fileCache.cacheDir, file.Name())
		data, err := ioutil.ReadFile(cachePath)
		if err != nil {
			continue
		}

		var entry CacheEntry
		decoder := gob.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&entry); err != nil {
			continue
		}

		// Classify cache entry by type
		switch entry.Data.(type) {
		case []byte:
			fileContentCount++
		case []string:
			treeSitterCount++
		case *models.ProjectSnapshot:
			snapshotCount++
		case *models.FullContextData:
			configCount++
		}
	}

	stats["cache_files"] = len(files)
	stats["total_size"] = totalSize
	stats["total_size_mb"] = float64(totalSize) / (1024 * 1024)
	stats["cache_dir"] = cm.fileCache.cacheDir
	stats["file_content_entries"] = fileContentCount
	stats["tree_sitter_entries"] = treeSitterCount
	stats["snapshot_entries"] = snapshotCount
	stats["config_entries"] = configCount

	if len(files) > 0 {
		stats["oldest_entry"] = oldestTime.Format(time.RFC3339)
		stats["newest_entry"] = newestTime.Format(time.RFC3339)
		stats["age_range_hours"] = newestTime.Sub(oldestTime).Hours()
	}

	return stats, nil
}

// CacheCleanupOptions defines options for cache cleanup
type CacheCleanupOptions struct {
	MaxAge   time.Duration // Remove entries older than this
	MaxSize  int64         // Remove oldest entries if cache exceeds this size (bytes)
	MaxFiles int           // Remove oldest entries if cache exceeds this number of files
	DryRun   bool          // If true, only report what would be cleaned without actual deletion
}

// SmartCleanupCache performs intelligent cache cleanup based on various criteria
func (cm *CacheManager) SmartCleanupCache(options CacheCleanupOptions) (map[string]interface{}, error) {
	cm.fileCache.mutex.Lock()
	defer cm.fileCache.mutex.Unlock()

	files, err := ioutil.ReadDir(cm.fileCache.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Collect file info with metadata
	type fileInfo struct {
		name     string
		path     string
		size     int64
		modTime  time.Time
		entryAge time.Time
	}

	var fileInfos []fileInfo
	var totalSize int64

	cutoffTime := time.Time{}
	if options.MaxAge > 0 {
		cutoffTime = time.Now().Add(-options.MaxAge)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		cachePath := filepath.Join(cm.fileCache.cacheDir, file.Name())

		// Try to read the cache entry to get its timestamp
		entryAge := file.ModTime() // Fallback to file modification time
		if data, err := ioutil.ReadFile(cachePath); err == nil {
			var entry CacheEntry
			if decoder := gob.NewDecoder(bytes.NewReader(data)); decoder.Decode(&entry) == nil {
				entryAge = entry.Timestamp
			}
		}

		fileInfos = append(fileInfos, fileInfo{
			name:     file.Name(),
			path:     cachePath,
			size:     file.Size(),
			modTime:  file.ModTime(),
			entryAge: entryAge,
		})
		totalSize += file.Size()
	}

	// Sort files by entry age (oldest first) for cleanup priority
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].entryAge.Before(fileInfos[j].entryAge)
	})

	var toDelete []fileInfo
	var deletedSize int64
	var deletedByAge, deletedBySize, deletedByCount int

	// Phase 1: Remove by age
	if !cutoffTime.IsZero() {
		for _, f := range fileInfos {
			if f.entryAge.Before(cutoffTime) {
				toDelete = append(toDelete, f)
				deletedSize += f.size
				deletedByAge++
			}
		}
	}

	// Phase 2: Remove by total size (oldest first)
	if options.MaxSize > 0 && totalSize > options.MaxSize {
		remainingFiles := make([]fileInfo, 0)
		for _, f := range fileInfos {
			// Skip files already marked for deletion by age
			alreadyMarked := false
			for _, d := range toDelete {
				if d.path == f.path {
					alreadyMarked = true
					break
				}
			}
			if !alreadyMarked {
				remainingFiles = append(remainingFiles, f)
			}
		}

		currentSize := totalSize - deletedSize
		for _, f := range remainingFiles {
			if currentSize <= options.MaxSize {
				break
			}
			toDelete = append(toDelete, f)
			deletedSize += f.size
			currentSize -= f.size
			deletedBySize++
		}
	}

	// Phase 3: Remove by file count (oldest first)
	if options.MaxFiles > 0 && len(fileInfos) > options.MaxFiles {
		remainingFiles := make([]fileInfo, 0)
		for _, f := range fileInfos {
			// Skip files already marked for deletion
			alreadyMarked := false
			for _, d := range toDelete {
				if d.path == f.path {
					alreadyMarked = true
					break
				}
			}
			if !alreadyMarked {
				remainingFiles = append(remainingFiles, f)
			}
		}

		excessCount := len(remainingFiles) - (options.MaxFiles - len(toDelete))
		for i := 0; i < excessCount && i < len(remainingFiles); i++ {
			f := remainingFiles[i]
			toDelete = append(toDelete, f)
			deletedSize += f.size
			deletedByCount++
		}
	}

	// Execute cleanup (or simulate if dry run)
	actuallyDeleted := 0
	if !options.DryRun {
		for _, f := range toDelete {
			if err := os.Remove(f.path); err == nil {
				actuallyDeleted++
			}
		}
	} else {
		actuallyDeleted = len(toDelete)
	}

	// Return cleanup summary
	result := map[string]interface{}{
		"files_before_cleanup":    len(fileInfos),
		"total_size_before_mb":    float64(totalSize) / (1024 * 1024),
		"files_marked_for_delete": len(toDelete),
		"size_to_delete_mb":       float64(deletedSize) / (1024 * 1024),
		"files_actually_deleted":  actuallyDeleted,
		"deleted_by_age":          deletedByAge,
		"deleted_by_size":         deletedBySize,
		"deleted_by_count":        deletedByCount,
		"files_after_cleanup":     len(fileInfos) - actuallyDeleted,
		"total_size_after_mb":     float64(totalSize-deletedSize) / (1024 * 1024),
		"dry_run":                 options.DryRun,
	}

	return result, nil
}

// performAutoCleanup performs background automatic cleanup with conservative defaults
func (cm *CacheManager) performAutoCleanup() {
	// Conservative cleanup: remove entries older than 7 days or if cache exceeds 100MB
	options := CacheCleanupOptions{
		MaxAge:   7 * 24 * time.Hour, // 7 days
		MaxSize:  100 * 1024 * 1024,  // 100MB
		MaxFiles: 1000,               // Max 1000 files
		DryRun:   false,
	}

	cm.SmartCleanupCache(options)
}

// ClearCache completely removes all cache entries
func (cm *CacheManager) ClearCache() error {
	cm.fileCache.mutex.Lock()
	defer cm.fileCache.mutex.Unlock()

	files, err := ioutil.ReadDir(cm.fileCache.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	var deletedCount int
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		cachePath := filepath.Join(cm.fileCache.cacheDir, file.Name())
		if err := os.Remove(cachePath); err == nil {
			deletedCount++
		}
	}

	return nil
}

// SetProjectSnapshot stores project snapshot data in cache without file system checks
func (cm *CacheManager) SetProjectSnapshot(key string, snapshot *models.ProjectSnapshot) error {
	cm.fileCache.mutex.Lock()
	defer cm.fileCache.mutex.Unlock()

	entry := CacheEntry{
		Data:      snapshot,
		Timestamp: time.Now(),
		FileSize:  0, // Not applicable for snapshots
		ModTime:   time.Now(),
		Hash:      key,
	}

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode snapshot entry: %w", err)
	}

	cacheKey := cm.fileCache.generateCacheKey(key)
	cachePath := cm.fileCache.getCachePath(cacheKey)

	if err := ioutil.WriteFile(cachePath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write snapshot cache file: %w", err)
	}

	return nil
}

// GetProjectSnapshot retrieves project snapshot data from cache
func (cm *CacheManager) GetProjectSnapshot(key string) (*models.ProjectSnapshot, bool) {
	cm.fileCache.mutex.RLock()
	defer cm.fileCache.mutex.RUnlock()

	cacheKey := cm.fileCache.generateCacheKey(key)
	cachePath := cm.fileCache.getCachePath(cacheKey)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, false
	}

	// Read cache file
	data, err := ioutil.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}

	// Decode the cache entry
	var entry CacheEntry
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&entry); err != nil {
		return nil, false
	}

	// Extract snapshot from entry
	if snapshot, ok := entry.Data.(*models.ProjectSnapshot); ok {
		return snapshot, true
	}

	return nil, false
}

// CleanExpiredCache removes cache entries older than specified duration
func (cm *CacheManager) CleanExpiredCache(maxAge time.Duration) error {
	cm.fileCache.mutex.Lock()
	defer cm.fileCache.mutex.Unlock()

	files, err := ioutil.ReadDir(cm.fileCache.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		cachePath := filepath.Join(cm.fileCache.cacheDir, file.Name())

		// Read cache entry to check timestamp
		data, err := ioutil.ReadFile(cachePath)
		if err != nil {
			continue
		}

		var entry CacheEntry
		decoder := gob.NewDecoder(bytes.NewReader(data))
		if err := decoder.Decode(&entry); err != nil {
			continue
		}

		// Remove if older than cutoff
		if entry.Timestamp.Before(cutoff) {
			os.Remove(cachePath)
		}
	}

	return nil
}

// GetFullCacheReport returns a comprehensive cache report including performance and storage stats
func (cm *CacheManager) GetFullCacheReport() (map[string]interface{}, error) {
	report := make(map[string]interface{})

	// Get performance stats
	perfStats := cm.GetPerformanceStats()
	report["performance"] = perfStats

	// Get detailed cache stats
	cacheStats, err := cm.GetDetailedCacheStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get detailed cache stats: %w", err)
	}
	report["storage"] = cacheStats

	// Calculate efficiency metrics
	efficiency := make(map[string]interface{})
	if totalFiles := cacheStats["cache_files"].(int); totalFiles > 0 {
		totalSizeMB := cacheStats["total_size_mb"].(float64)
		avgFileSizeKB := (totalSizeMB * 1024) / float64(totalFiles)
		efficiency["avg_file_size_kb"] = avgFileSizeKB
		efficiency["storage_efficiency"] = "good"
		if avgFileSizeKB > 100 {
			efficiency["storage_efficiency"] = "check large files"
		}
	}

	if hitRate := perfStats["hit_rate"].(float64); hitRate > 0 {
		efficiency["cache_efficiency"] = "excellent"
		if hitRate < 50 {
			efficiency["cache_efficiency"] = "poor - consider cache warming"
		} else if hitRate < 75 {
			efficiency["cache_efficiency"] = "moderate"
		}
	}

	report["efficiency"] = efficiency
	report["generated_at"] = time.Now().Format(time.RFC3339)

	return report, nil
}
