package browser

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// ScriptCache 脚本缓存管理器
// 缓存生成的 stealth 脚本，避免重复生成
type ScriptCache struct {
	cache   map[string]*cachedScript
	mutex   sync.RWMutex
	maxAge  time.Duration // 缓存过期时间
	maxSize int           // 最大缓存数量
}

// cachedScript 缓存的脚本
type cachedScript struct {
	script    string
	hash      string
	createdAt time.Time
	hitCount  int64
}

var (
	// 全局脚本缓存实例
	globalScriptCache *ScriptCache
	scriptCacheOnce   sync.Once

	// 静态脚本缓存（只生成一次）
	advancedStealthScript     string
	advancedStealthScriptOnce sync.Once
	simpleStealthScript       string
	simpleStealthScriptOnce   sync.Once
	baseStealthScript         string
	baseStealthScriptOnce     sync.Once
)

// GetScriptCache 获取全局脚本缓存实例
func GetScriptCache() *ScriptCache {
	scriptCacheOnce.Do(func() {
		globalScriptCache = NewScriptCache(100, 30*time.Minute)
	})
	return globalScriptCache
}

// NewScriptCache 创建新的脚本缓存
func NewScriptCache(maxSize int, maxAge time.Duration) *ScriptCache {
	sc := &ScriptCache{
		cache:   make(map[string]*cachedScript),
		maxAge:  maxAge,
		maxSize: maxSize,
	}
	// 启动后台清理协程
	go sc.cleanupLoop()
	return sc
}

// GetOrGenerate 获取缓存的脚本，如果不存在则生成
func (sc *ScriptCache) GetOrGenerate(key string, generator func() string) string {
	// 快速路径：读取缓存
	sc.mutex.RLock()
	if cached, ok := sc.cache[key]; ok {
		if time.Since(cached.createdAt) < sc.maxAge {
			cached.hitCount++
			sc.mutex.RUnlock()
			return cached.script
		}
	}
	sc.mutex.RUnlock()

	// 慢速路径：生成脚本
	script := generator()
	hash := fmt.Sprintf("%x", md5.Sum([]byte(script)))

	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// 双重检查
	if cached, ok := sc.cache[key]; ok {
		if time.Since(cached.createdAt) < sc.maxAge {
			cached.hitCount++
			return cached.script
		}
	}

	// 检查是否需要淘汰
	if len(sc.cache) >= sc.maxSize {
		sc.evictOldest()
	}

	// 存储新脚本
	sc.cache[key] = &cachedScript{
		script:    script,
		hash:      hash,
		createdAt: time.Now(),
		hitCount:  1,
	}

	return script
}

// evictOldest 淘汰最旧的缓存项（需要持有写锁）
func (sc *ScriptCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range sc.cache {
		if oldestKey == "" || cached.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.createdAt
		}
	}

	if oldestKey != "" {
		delete(sc.cache, oldestKey)
	}
}

// cleanupLoop 定期清理过期缓存
func (sc *ScriptCache) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		sc.cleanup()
	}
}

// cleanup 清理过期缓存
func (sc *ScriptCache) cleanup() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	now := time.Now()
	for key, cached := range sc.cache {
		if now.Sub(cached.createdAt) > sc.maxAge {
			delete(sc.cache, key)
		}
	}
}

// Stats 返回缓存统计信息
func (sc *ScriptCache) Stats() map[string]interface{} {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	var totalHits int64
	for _, cached := range sc.cache {
		totalHits += cached.hitCount
	}

	return map[string]interface{}{
		"size":      len(sc.cache),
		"maxSize":   sc.maxSize,
		"totalHits": totalHits,
	}
}

// Clear 清空缓存
func (sc *ScriptCache) Clear() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.cache = make(map[string]*cachedScript)
}

// ========== 静态脚本缓存函数 ==========

// GetCachedAdvancedStealthScript 获取缓存的高级 stealth 脚本
// 只在首次调用时生成，之后直接返回缓存
func GetCachedAdvancedStealthScript() string {
	advancedStealthScriptOnce.Do(func() {
		advancedStealthScript = generateAdvancedStealthScript()
	})
	return advancedStealthScript
}

// GetCachedSimpleStealthScript 获取缓存的简单 stealth 脚本
func GetCachedSimpleStealthScript() string {
	simpleStealthScriptOnce.Do(func() {
		simpleStealthScript = generateSimpleStealthScript()
	})
	return simpleStealthScript
}

// GetCachedBaseStealthScript 获取缓存的基础 stealth 脚本
func GetCachedBaseStealthScript() string {
	baseStealthScriptOnce.Do(func() {
		baseStealthScript = generateBaseStealthScript()
	})
	return baseStealthScript
}

// GetCachedStealthScriptWithConfig 获取基于配置的缓存脚本
// 使用 userID 作为缓存键
func GetCachedStealthScriptWithConfig(config *FingerprintConfig) string {
	if config == nil {
		return GetCachedAdvancedStealthScript()
	}

	// 使用 userID 作为缓存键
	cacheKey := config.UserID
	if cacheKey == "" {
		// 没有 userID，不缓存
		return generateStealthScriptWithConfig(config)
	}

	cache := GetScriptCache()
	return cache.GetOrGenerate(cacheKey, func() string {
		return generateStealthScriptWithConfig(config)
	})
}

// ========== 内部生成函数 ==========

// generateAdvancedStealthScript 生成高级 stealth 脚本（内部函数）
func generateAdvancedStealthScript() string {
	return GetAdvancedStealthScript()
}

// generateSimpleStealthScript 生成简单 stealth 脚本（内部函数）
func generateSimpleStealthScript() string {
	return GetSimpleStealthScript()
}

// generateBaseStealthScript 生成基础 stealth 脚本（内部函数）
func generateBaseStealthScript() string {
	return GetBaseStealthScript()
}

// generateStealthScriptWithConfig 生成配置化脚本（内部函数）
func generateStealthScriptWithConfig(config *FingerprintConfig) string {
	return GetStealthScriptWithConfig(config)
}
