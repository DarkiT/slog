package cachekey

import (
	"strconv"
	"strings"
	"unsafe"

	"github.com/darkit/slog/internal/xxhash"
)

// CacheKey 缓存键优化器
type CacheKey struct {
	useXXHash   bool
	prefixCache map[string]uint64
}

// New 创建缓存键优化器
func New() *CacheKey {
	return &CacheKey{
		useXXHash:   true,
		prefixCache: make(map[string]uint64),
	}
}

// GenerateKey 生成优化的缓存键
// 使用 xxhash 算法生成高性能、低碰撞的缓存键
func (cko *CacheKey) GenerateKey(prefix, data string) string {
	if !cko.useXXHash {
		return prefix + ":" + data
	}

	if len(data) <= 32 {
		return prefix + ":" + data
	}

	hash := xxhash.Sum64String(data)
	return prefix + ":h" + strconv.FormatUint(hash, 16) + ":" + strconv.Itoa(len(data))
}

// GenerateKeyWithContext 生成带上下文的缓存键
func (cko *CacheKey) GenerateKeyWithContext(desensitizer, dataType, data string) string {
	if !cko.useXXHash {
		return desensitizer + ":" + dataType + ":" + data
	}

	if len(data) <= 32 {
		return desensitizer + ":" + dataType + ":" + data
	}

	contextHash := cko.getOrComputeContextHash(desensitizer + ":" + dataType)
	dataHash := xxhash.Sum64String(data)
	combinedHash := contextHash ^ dataHash

	return "ch:" + strconv.FormatUint(combinedHash, 16) + ":" + strconv.Itoa(len(data))
}

func (cko *CacheKey) getOrComputeContextHash(context string) uint64 {
	if hash, exists := cko.prefixCache[context]; exists {
		return hash
	}

	hash := xxhash.Sum64String(context)

	if len(cko.prefixCache) < 100 {
		cko.prefixCache[context] = hash
	}

	return hash
}

// GenerateHashKey 生成纯哈希键
func (cko *CacheKey) GenerateHashKey(data string) string {
	if !cko.useXXHash {
		return data
	}

	if len(data) <= 16 {
		return data
	}

	hash := xxhash.Sum64String(data)
	return strconv.FormatUint(hash, 36)
}

// GenerateFastKey 生成快速键（平衡性能和碰撞率）
func (cko *CacheKey) GenerateFastKey(data string) string {
	dataLen := len(data)

	if dataLen <= 32 {
		return data
	}

	if !cko.useXXHash {
		prefixLen := 8
		suffixLen := 8
		if dataLen < prefixLen+suffixLen {
			return data
		}
		return data[:prefixLen] + "..." + data[dataLen-suffixLen:] + ":" + strconv.Itoa(dataLen)
	}

	prefixLen := 8
	if dataLen < prefixLen {
		prefixLen = dataLen
	}

	middlePart := data[prefixLen:]
	hash := xxhash.Sum64String(middlePart)

	return data[:prefixLen] + "h" + strconv.FormatUint(hash, 16) + ":" + strconv.Itoa(dataLen)
}

// GenerateLayeredKey 生成分层键
func (cko *CacheKey) GenerateLayeredKey(layers ...string) string {
	if len(layers) == 0 {
		return ""
	}

	if len(layers) == 1 {
		return cko.GenerateHashKey(layers[0])
	}

	var keyBuilder strings.Builder
	keyBuilder.Grow(64)

	for i := 0; i < len(layers)-1; i++ {
		if i > 0 {
			keyBuilder.WriteByte(':')
		}
		keyBuilder.WriteString(layers[i])
	}

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
		keyBuilder.WriteByte(':')
		keyBuilder.WriteString(data)
	}

	return keyBuilder.String()
}

// SetXXHashEnabled 设置是否启用 xxhash
func (cko *CacheKey) SetXXHashEnabled(enabled bool) {
	cko.useXXHash = enabled
	if !enabled {
		cko.prefixCache = make(map[string]uint64)
	}
}

// IsXXHashEnabled 是否启用 xxhash
func (cko *CacheKey) IsXXHashEnabled() bool {
	return cko.useXXHash
}

// GetCacheStats 获取缓存优化器统计信息
func (cko *CacheKey) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"xxhash_enabled":    cko.useXXHash,
		"prefix_cache_size": len(cko.prefixCache),
		"algorithm":         "xxhash64",
	}
}

// ClearPrefixCache 清除前缀缓存
func (cko *CacheKey) ClearPrefixCache() {
	cko.prefixCache = make(map[string]uint64)
}

// FastStringHash 快速字符串哈希（内联优化版本）
func (cko *CacheKey) FastStringHash(s string) uint64 {
	if !cko.useXXHash {
		var hash uint64 = 14695981039346656037
		for i := 0; i < len(s); i++ {
			hash ^= uint64(s[i])
			hash *= 1099511628211
		}
		return hash
	}

	return xxhash.Sum64(*(*[]byte)(unsafe.Pointer(&struct {
		string
		int
	}{s, len(s)})))
}

// HashCombine 组合多个哈希值
func (cko *CacheKey) HashCombine(hashes ...uint64) uint64 {
	if len(hashes) == 0 {
		return 0
	}
	if len(hashes) == 1 {
		return hashes[0]
	}

	result := hashes[0]
	for i := 1; i < len(hashes); i++ {
		result ^= hashes[i] + 0x9e3779b9 + (result << 6) + (result >> 2)
	}
	return result
}

// 全局缓存键优化器实例（仅供内部模块复用）
var Global = New()

// Key 全局键生成函数
func Key(prefix, data string) string {
	return Global.GenerateKey(prefix, data)
}

// KeyWithContext 全局上下文键生成函数
func KeyWithContext(desensitizer, dataType, data string) string {
	return Global.GenerateKeyWithContext(desensitizer, dataType, data)
}

// FastKey 全局快速键生成函数
func FastKey(data string) string {
	return Global.GenerateFastKey(data)
}
