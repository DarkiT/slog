package dlp

import (
	"strconv"
	"strings"
	"unsafe"

	"github.com/cespare/xxhash/v2"
)

// CacheKeyOptimizer 缓存键优化器
type CacheKeyOptimizer struct {
	useXXHash   bool
	prefixCache map[string]uint64
}

// NewCacheKeyOptimizer 创建缓存键优化器
func NewCacheKeyOptimizer() *CacheKeyOptimizer {
	return &CacheKeyOptimizer{
		useXXHash:   true,
		prefixCache: make(map[string]uint64),
	}
}

// GenerateKey 生成优化的缓存键
// 使用xxhash算法生成高性能、低碰撞的缓存键
func (cko *CacheKeyOptimizer) GenerateKey(prefix, data string) string {
	if !cko.useXXHash {
		// 回退到简单拼接方式
		return prefix + ":" + data
	}

	// 对于短数据，直接使用原始字符串作为键
	if len(data) <= 32 {
		return prefix + ":" + data
	}

	// 对于长数据，使用xxhash生成紧凑的键
	hash := xxhash.Sum64String(data)

	// 构建高效的缓存键：prefix + hash + 数据长度
	// 添加数据长度可以进一步减少碰撞概率
	return prefix + ":h" + strconv.FormatUint(hash, 16) + ":" + strconv.Itoa(len(data))
}

// GenerateKeyWithContext 生成带上下文的缓存键
// 包含数据类型和脱敏器类型信息，提供更强的唯一性保证
func (cko *CacheKeyOptimizer) GenerateKeyWithContext(desensitizer, dataType, data string) string {
	if !cko.useXXHash {
		return desensitizer + ":" + dataType + ":" + data
	}

	// 对于短数据，使用完整信息作为键
	if len(data) <= 32 {
		return desensitizer + ":" + dataType + ":" + data
	}

	// 对于长数据，使用组合哈希
	// 1. 首先对脱敏器和数据类型组合进行哈希
	contextHash := cko.getOrComputeContextHash(desensitizer + ":" + dataType)

	// 2. 对数据内容进行哈希
	dataHash := xxhash.Sum64String(data)

	// 3. 组合两个哈希值，生成最终键
	// 使用XOR组合避免简单的字符串拼接开销
	combinedHash := contextHash ^ dataHash

	return "ch:" + strconv.FormatUint(combinedHash, 16) + ":" + strconv.Itoa(len(data))
}

// getOrComputeContextHash 获取或计算上下文哈希（带缓存）
func (cko *CacheKeyOptimizer) getOrComputeContextHash(context string) uint64 {
	if hash, exists := cko.prefixCache[context]; exists {
		return hash
	}

	hash := xxhash.Sum64String(context)

	// 限制缓存大小，避免内存泄漏
	if len(cko.prefixCache) < 100 {
		cko.prefixCache[context] = hash
	}

	return hash
}

// GenerateHashKey 生成纯哈希键（最高性能）
// 适用于不需要可读性的场景，提供最佳性能
func (cko *CacheKeyOptimizer) GenerateHashKey(data string) string {
	if !cko.useXXHash {
		return data
	}

	// 对于很短的字符串，直接返回
	if len(data) <= 16 {
		return data
	}

	hash := xxhash.Sum64String(data)
	return strconv.FormatUint(hash, 36) // 使用36进制获得更短的字符串
}

// GenerateFastKey 生成快速键（平衡性能和碰撞率）
// 使用前缀+后缀+长度+哈希的组合策略
func (cko *CacheKeyOptimizer) GenerateFastKey(data string) string {
	dataLen := len(data)

	if dataLen <= 32 {
		return data
	}

	if !cko.useXXHash {
		// 回退策略：前缀+后缀+长度
		prefixLen := 8
		suffixLen := 8
		if dataLen < prefixLen+suffixLen {
			return data
		}
		return data[:prefixLen] + "..." + data[dataLen-suffixLen:] + ":" + strconv.Itoa(dataLen)
	}

	// 高性能策略：前缀+哈希+长度
	prefixLen := 8
	if dataLen < prefixLen {
		prefixLen = dataLen
	}

	// 使用xxhash对中间部分进行哈希
	middlePart := data[prefixLen:]
	hash := xxhash.Sum64String(middlePart)

	return data[:prefixLen] + "h" + strconv.FormatUint(hash, 16) + ":" + strconv.Itoa(dataLen)
}

// GenerateLayeredKey 生成分层键（适用于复杂场景）
// 支持多级前缀和参数，提供最大的灵活性
func (cko *CacheKeyOptimizer) GenerateLayeredKey(layers ...string) string {
	if len(layers) == 0 {
		return ""
	}

	if len(layers) == 1 {
		return cko.GenerateHashKey(layers[0])
	}

	// 构建分层键
	var keyBuilder strings.Builder
	keyBuilder.Grow(64) // 预分配容量

	// 前n-1层作为前缀
	for i := 0; i < len(layers)-1; i++ {
		if i > 0 {
			keyBuilder.WriteByte(':')
		}
		keyBuilder.WriteString(layers[i])
	}

	// 最后一层作为数据部分
	data := layers[len(layers)-1]

	if len(data) <= 32 {
		keyBuilder.WriteByte(':')
		keyBuilder.WriteString(data)
	} else if cko.useXXHash {
		hash := xxhash.Sum64String(data)
		keyBuilder.WriteString(":h")
		keyBuilder.WriteString(strconv.FormatUint(hash, 16))
		keyBuilder.WriteByte(':')
		keyBuilder.WriteString(strconv.Itoa(len(data)))
	} else {
		// 回退策略
		keyBuilder.WriteByte(':')
		keyBuilder.WriteString(data)
	}

	return keyBuilder.String()
}

// SetXXHashEnabled 设置是否启用xxhash
func (cko *CacheKeyOptimizer) SetXXHashEnabled(enabled bool) {
	cko.useXXHash = enabled
	if !enabled {
		// 清除前缀缓存以节省内存
		cko.prefixCache = make(map[string]uint64)
	}
}

// IsXXHashEnabled 检查是否启用xxhash
func (cko *CacheKeyOptimizer) IsXXHashEnabled() bool {
	return cko.useXXHash
}

// GetCacheStats 获取缓存优化器统计信息
func (cko *CacheKeyOptimizer) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"xxhash_enabled":    cko.useXXHash,
		"prefix_cache_size": len(cko.prefixCache),
		"algorithm":         "xxhash64",
	}
}

// ClearPrefixCache 清除前缀缓存
func (cko *CacheKeyOptimizer) ClearPrefixCache() {
	cko.prefixCache = make(map[string]uint64)
}

// FastStringHash 快速字符串哈希（内联优化版本）
// 使用unsafe提供最高性能的哈希计算
func (cko *CacheKeyOptimizer) FastStringHash(s string) uint64 {
	if !cko.useXXHash {
		// 简单的FNV哈希作为回退
		var hash uint64 = 14695981039346656037
		for i := 0; i < len(s); i++ {
			hash ^= uint64(s[i])
			hash *= 1099511628211
		}
		return hash
	}

	// 使用unsafe优化的xxhash
	return xxhash.Sum64(*(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)})))
}

// HashCombine 组合多个哈希值
// 使用boost风格的哈希组合算法
func (cko *CacheKeyOptimizer) HashCombine(hashes ...uint64) uint64 {
	if len(hashes) == 0 {
		return 0
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	result := hashes[0]
	for i := 1; i < len(hashes); i++ {
		// boost::hash_combine算法
		result ^= hashes[i] + 0x9e3779b9 + (result << 6) + (result >> 2)
	}
	return result
}

// 全局缓存键优化器实例
var globalCacheOptimizer = NewCacheKeyOptimizer()

// GetGlobalCacheOptimizer 获取全局缓存键优化器
func GetGlobalCacheOptimizer() *CacheKeyOptimizer {
	return globalCacheOptimizer
}

// OptimizeKey 全局键优化函数（便利函数）
func OptimizeKey(prefix, data string) string {
	return globalCacheOptimizer.GenerateKey(prefix, data)
}

// OptimizeKeyWithContext 全局上下文键优化函数
func OptimizeKeyWithContext(desensitizer, dataType, data string) string {
	return globalCacheOptimizer.GenerateKeyWithContext(desensitizer, dataType, data)
}

// OptimizeFastKey 全局快速键优化函数
func OptimizeFastKey(data string) string {
	return globalCacheOptimizer.GenerateFastKey(data)
}
