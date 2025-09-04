package code_analyzer

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/meysamhadeli/codai/code_analyzer/models"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	FileSize  int64       `json:"file_size"`
	ModTime   time.Time   `json:"mod_time"`
	Hash      string      `json:"hash"`
}

// FileCache manages file-based caching with intelligent invalidation
type FileCache struct {
	cacheDir string
	mutex    sync.RWMutex
}

// CacheManager provides high-level caching operations
type CacheManager struct {
	fileCache *FileCache
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(cacheDir string) (*CacheManager, error) {
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".codai", "cache")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	fileCache := &FileCache{
		cacheDir: cacheDir,
	}

	return &CacheManager{
		fileCache: fileCache,
	}, nil
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
	if err := json.Unmarshal(data, &entry); err != nil {
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

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	cacheKey := fc.generateCacheKey(filePath)
	cachePath := fc.getCachePath(cacheKey)

	if err := ioutil.WriteFile(cachePath, jsonData, 0644); err != nil {
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
		return nil, false
	}

	// Type assertion to convert back to FullContextData
	if contextData, ok := data.(*models.FullContextData); ok {
		return contextData, true
	}

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
		return nil, false
	}

	if content, ok := data.([]byte); ok {
		return content, true
	}

	return nil, false
}

// SetFileContentCache stores file content in cache
func (cm *CacheManager) SetFileContentCache(filePath string, content []byte) error {
	return cm.fileCache.Set(filePath, content)
}

// GetTreeSitterCache retrieves cached tree-sitter parsing results
func (cm *CacheManager) GetTreeSitterCache(filePath string) ([]string, bool) {
	data, found := cm.fileCache.Get(filePath + ".treesitter")
	if !found {
		return nil, false
	}

	if codeParts, ok := data.([]string); ok {
		return codeParts, true
	}

	return nil, false
}

// SetTreeSitterCache stores tree-sitter parsing results in cache
func (cm *CacheManager) SetTreeSitterCache(filePath string, codeParts []string) error {
	return cm.fileCache.Set(filePath+".treesitter", codeParts)
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
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}

		// Remove if older than cutoff
		if entry.Timestamp.Before(cutoff) {
			os.Remove(cachePath)
		}
	}

	return nil
}