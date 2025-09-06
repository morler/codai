package code_analyzer

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/zeebo/xxh3"
)

// BenchmarkCacheKeyGeneration 性能测试
func BenchmarkCacheKeyGeneration(b *testing.B) {
	// 生成测试用的随机文件路径
	filePaths := make([]string, 1000)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789/_-."
	for i := 0; i < 1000; i++ {
		length := rand.Intn(100) + 20 // 20-119字符长度
		path := ""
		for j := 0; j < length; j++ {
			path += string(charset[rand.Intn(len(charset))])
		}
		filePaths[i] = path
	}

	b.Run("MD5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := filePaths[i%1000]
			hash := md5.Sum([]byte(path))
			_ = fmt.Sprintf("%x.cache", hash)
		}
	})

	b.Run("XXH3", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := filePaths[i%1000]
			hash := xxh3.HashString(path)
			_ = fmt.Sprintf("%x.cache", hash)
		}
	})
}

// BenchmarkRealWorldFilePaths 使用实际文件路径的基准测试
func BenchmarkRealWorldFilePaths(b *testing.B) {
	// 实际项目中常见的文件路径
	realPaths := []string{
		"code_analyzer/cache.go",
		"providers/openai/openai_provider.go",
		"providers/anthropic/anthropic_provider.go",
		"embed_data/tree-sitter/queries/javascript.scm",
		"cmd/code.go",
		"config/config.go",
		"utils/markdown_generator.go",
		"CLAUDE.md",
		"README.md",
		"Makefile",
		"go.mod",
		"main.go",
		"codai-config.yml",
		".github/workflows/ci.yml",
		"assets/codai-demo.gif",
		"long/path/to/some/deeply/nested/file/in/a/big/project/structure.go",
		"src/components/ui/button/component.tsx",
		"test/analyzer/analyzer_test.go",
		"vendor/github.com/spf13/cobra/command.go",
	}

	b.Run("MD5_RealPaths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := realPaths[i%len(realPaths)]
			hash := md5.Sum([]byte(path))
			_ = fmt.Sprintf("%x.cache", hash)
		}
	})

	b.Run("XXH3_RealPaths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			path := realPaths[i%len(realPaths)]
			hash := xxh3.HashString(path)
			_ = fmt.Sprintf("%x.cache", hash)
		}
	})
}

// TestXXH3CacheKeyConsistency 确保XXH3算法的一致性
func TestXXH3CacheKeyConsistency(t *testing.T) {
	path := "code_analyzer/cache.go"
	
	// 多次调用应该返回相同的结果
	for i := 0; i < 100; i++ {
		hash1 := xxh3.HashString(path)
		cacheKey1 := fmt.Sprintf("%x.cache", hash1)
		
		hash2 := xxh3.HashString(path)
		cacheKey2 := fmt.Sprintf("%x.cache", hash2)
		
		if cacheKey1 != cacheKey2 {
			t.Errorf("XXH3 hash inconsistency: %s != %s", cacheKey1, cacheKey2)
		}
	}
}

// TestPerformanceImprovementAnalysis 分析性能提升
func TestPerformanceImprovementAnalysis(t *testing.T) {
	// 模拟实际文件路径
	paths := []string{
		"code_analyzer/cache.go",
		"providers/openai/openai_provider.go", 
		"cmd/code.go",
	}

	const iterations = 1000000

	start := time.Now()
	for i := 0; i < iterations; i++ {
		path := paths[i%len(paths)]
		hash := md5.Sum([]byte(path))
		hashKey := fmt.Sprintf("%x.cache", hash)
		_ = hashKey
	}
	md5Duration := time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		path := paths[i%len(paths)]
		hash := xxh3.HashString(path)
		hashKey := fmt.Sprintf("%x.cache", hash)
		_ = hashKey
	}
	xxh3Duration := time.Since(start)

	fmt.Printf("MD5 运算耗时: %v\n", md5Duration)
	fmt.Printf("XXH3 运算耗时: %v\n", xxh3Duration)
	
	if md5Duration > 0 && xxh3Duration > 0 {
		improvement := float64(md5Duration) / float64(xxh3Duration)
		improvementPercent := (improvement - 1) * 100
		fmt.Printf("XXH3比MD5快了: %.2f倍 (%.1f%%性能提升)\n", improvement, improvementPercent)
	}
}