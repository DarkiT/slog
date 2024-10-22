package slog

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// levelManager 管理动态日志级别更新
type levelManager struct {
	mu            sync.RWMutex
	levelUpdateCh chan Level
	observers     map[string]func(Level)
	done          chan struct{}
}

var (
	manager     *levelManager
	managerOnce sync.Once
)

// 获取levelManager单例
func getLevelManager() *levelManager {
	managerOnce.Do(func() {
		manager = &levelManager{
			levelUpdateCh: make(chan Level, 10), // 缓冲通道，避免阻塞
			observers:     make(map[string]func(Level)),
			done:          make(chan struct{}),
		}
		go manager.run()
	})
	return manager
}

// run 运行级别更新处理循环
func (m *levelManager) run() {
	ticker := time.NewTicker(100 * time.Millisecond) // 定期检查更新
	defer ticker.Stop()

	for {
		select {
		case level := <-m.levelUpdateCh:
			m.updateLevel(level)
		case <-m.done:
			return
		case <-ticker.C:
			// 可以在这里添加额外的检查逻辑
		}
	}
}

// updateLevel 更新日志级别
func (m *levelManager) updateLevel(level Level) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 更新全局级别
	levelVar.Set(level)

	// 通知所有观察者
	for _, observer := range m.observers {
		if observer != nil {
			observer(level)
		}
	}
}

// RegisterObserver 注册一个观察者来监听级别变化
func (m *levelManager) RegisterObserver(name string, observer func(Level)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.observers[name] = observer
}

// UnregisterObserver 注销一个观察者
func (m *levelManager) UnregisterObserver(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.observers, name)
}

// Close 关闭级别管理器
func (m *levelManager) Close() {
	close(m.done)
}

// UpdateLogLevel 动态更新日志级别
// level 可以是数字(-8, -4, 0, 4, 8, 12)或字符串(trace, debug, info, warn, error, fatal)
func UpdateLogLevel(level any) error {
	var newLevel Level

	switch v := level.(type) {
	case Level:
		newLevel = v
	case int:
		newLevel = Level(v)
	case string:
		// 将字符串转换为Level
		switch strings.ToLower(v) {
		case "trace":
			newLevel = LevelTrace
		case "debug":
			newLevel = LevelDebug
		case "info":
			newLevel = LevelInfo
		case "warn":
			newLevel = LevelWarn
		case "error":
			newLevel = LevelError
		case "fatal":
			newLevel = LevelFatal
		default:
			return errors.New("invalid log level string")
		}
	default:
		return errors.New("unsupported level type")
	}

	// 验证级别是否有效
	if !isValidLevel(newLevel) {
		return errors.New("invalid log level value")
	}

	// 发送更新请求
	select {
	case getLevelManager().levelUpdateCh <- newLevel:
		return nil
	case <-time.After(time.Second):
		return errors.New("update level timeout")
	}
}

// isValidLevel 检查日志级别是否有效
func isValidLevel(level Level) bool {
	validLevels := []Level{
		LevelTrace, // -8
		LevelDebug, // -4
		LevelInfo,  // 0
		LevelWarn,  // 4
		LevelError, // 8
		LevelFatal, // 12
	}

	for _, l := range validLevels {
		if level == l {
			return true
		}
	}
	return false
}

// WatchLevel 观察日志级别变化
func WatchLevel(name string, callback func(Level)) {
	getLevelManager().RegisterObserver(name, callback)
}

// UnwatchLevel 取消观察日志级别变化
func UnwatchLevel(name string) {
	getLevelManager().UnregisterObserver(name)
}

// 在包初始化时启动levelManager
func init() {
	getLevelManager()
}
