package browser

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// TimestampFingerprintInjector TS1 时间戳指纹注入器
// 修改时间相关的API以产生不同的时间戳指纹
type TimestampFingerprintInjector struct {
	config      *FingerprintConfig
	userHash    string
	timeOffset  int64   // 毫秒级时间偏移
	perfOffset  float64 // performance.now() 偏移
	datePattern int     // 日期模式
	tzOffset    int     // 时区偏移分钟数
}

// NewTimestampFingerprintInjector 创建时间戳指纹注入器
func NewTimestampFingerprintInjector(config *FingerprintConfig) *TimestampFingerprintInjector {
	userHash := generateTimestampHash(config.UserID)

	// 基于用户哈希生成时间偏移
	seed1 := hashToInt(userHash, 0) % 1000000
	seed2 := hashToInt(userHash, 4) % 1000000

	return &TimestampFingerprintInjector{
		config:      config,
		userHash:    userHash,
		timeOffset:  int64(seed1%3000 - 1500),  // -1500ms 到 +1500ms
		perfOffset:  float64(seed2%100) / 10.0, // 0-10ms
		datePattern: hashToInt(userHash, 8) % 10, // 0-9 的模式
		tzOffset:    config.Timezone.Offset,      // 使用配置的时区偏移
	}
}

// GenerateTimestampInjectionScript 生成时间戳修改脚本
// 简化版本：只修改 Date.now() 和 Date 构造函数，不修改原型方法
func (t *TimestampFingerprintInjector) GenerateTimestampInjectionScript() string {
	return fmt.Sprintf(`
// ========================================
// TS1 时间戳指纹修改脚本 (简化版)
// 用户ID: %s
// 哈希: %s
// 策略: 只修改 Date.now() 和构造函数，不修改原型方法
// ========================================
(function() {
    'use strict';
    
    const userHash = '%s';
    const timeOffsetMs = %d;
    const perfOffsetMs = %.3f;
    const datePattern = %d;
    const tzOffsetMinutes = %d;
    
    // ====== 保存原始 Date ======
    const OriginalDate = window.Date;
    const originalDateNow = OriginalDate.now.bind(OriginalDate);
    const originalDateParse = OriginalDate.parse.bind(OriginalDate);
    const originalDateUTC = OriginalDate.UTC.bind(OriginalDate);
    
    // ====== 创建新的 Date 构造函数 ======
    function ModifiedDate(...args) {
        // 如果作为普通函数调用
        if (!new.target) {
            return new OriginalDate().toString();
        }
        
        // 如果无参数，使用当前时间 + 偏移
        if (args.length === 0) {
            const now = originalDateNow() + timeOffsetMs;
            return new OriginalDate(now);
        }
        
        // 如果有参数，直接传递给原始 Date
        return new OriginalDate(...args);
    }
    
    // 复制原型
    ModifiedDate.prototype = OriginalDate.prototype;
    
    // 创建修改后的 Date.now
    ModifiedDate.now = function() {
        return originalDateNow() + timeOffsetMs;
    };
    
    // 复制其他静态方法
    ModifiedDate.parse = originalDateParse;
    ModifiedDate.UTC = originalDateUTC;
    
    // 替换全局 Date
    window.Date = ModifiedDate;
    
    // ====== 修改 Performance API ======
    if (window.performance && window.performance.now) {
        const originalPerfNow = window.performance.now.bind(window.performance);
        const perfStartTime = originalPerfNow();
        
        window.performance.now = function() {
            return originalPerfNow() - perfStartTime + perfOffsetMs;
        };
    }
    
    console.log('✅ TS1 时间戳指纹修改已应用 (简化版)', {
        userHash: userHash.substr(0, 8) + '...',
        timeOffset: timeOffsetMs + 'ms',
        perfOffset: perfOffsetMs.toFixed(3) + 'ms'
    });
})();
`,
		t.config.UserID,
		t.userHash[:16]+"...",
		t.userHash,
		t.timeOffset,
		t.perfOffset,
		t.datePattern,
		t.tzOffset)
}

// generateTimestampHash 生成时间戳特定的哈希
func generateTimestampHash(userID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(userID + "_timestamp_fingerprint"))
	return hex.EncodeToString(hasher.Sum(nil))
}

// GetDebugInfo 获取调试信息
func (t *TimestampFingerprintInjector) GetDebugInfo() map[string]interface{} {
	return map[string]interface{}{
		"user_id":      t.config.UserID,
		"user_hash":    t.userHash[:16] + "...",
		"time_offset":  fmt.Sprintf("%dms", t.timeOffset),
		"perf_offset":  fmt.Sprintf("%.3fms", t.perfOffset),
		"date_pattern": t.datePattern,
		"timezone":     t.config.Timezone.Timezone,
		"tz_offset":    fmt.Sprintf("%d minutes", t.tzOffset),
	}
}

// CalculateExpectedTimestampHash 计算预期的时间戳哈希
func (t *TimestampFingerprintInjector) CalculateExpectedTimestampHash() string {
	data := fmt.Sprintf("%s_%d_%f_%d_%s",
		t.config.UserID,
		t.timeOffset,
		t.perfOffset,
		t.datePattern,
		t.config.Timezone.Timezone)

	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))[:40]
}

// CombineWithOtherScripts 与其他脚本结合
func (t *TimestampFingerprintInjector) CombineWithOtherScripts(baseScript, audioWebGLScript string) string {
	timestampScript := t.GenerateTimestampInjectionScript()

	return fmt.Sprintf(`
(() => {
    'use strict';
    
    console.log('🕐 开始注入完整指纹修改（包括时间戳）...');
    
    // 1. 基础隐身脚本
    %s
    
    // 2. Audio/WebGL 指纹修改
    %s
    
    // 3. 时间戳指纹修改
    %s
    
    console.log('✅ 所有指纹修改已完成（包括TS1）！用户ID: %s');
})();
`, baseScript, audioWebGLScript, timestampScript, t.config.UserID)
}
