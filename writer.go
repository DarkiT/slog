package slog

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	backupTimeFormat  = "2006-01-02T15-04-05"
	compressSuffix    = "gz"
	defaultMaxSize    = 100 // 默认单个文件最大 100MB
	defaultMaxAge     = 30  // 默认保留 30 天
	defaultMaxBackups = 30  // 默认保留 30 个备份文件
)

var _ io.WriteCloser = (*writer)(nil)

type writer struct {
	_Filename   string // 文件名称
	_MaxSize    int    // MB为单位
	_MaxAge     int    // 天数
	_MaxBackups int    // 最大备份数
	_LocalTime  bool
	_Compress   bool

	size int64
	file *os.File
	mu   sync.Mutex
}

type logInfo struct {
	timestamp time.Time
	name      string
}

// NewWriter 创建一个新的日志写入器,支持指定一个或多个文件路径,多个路径时使用第一个有效路径
// filename: 日志文件路径
// 默认配置:
//   - 单个文件最大 100MB
//   - 保留最近 30 天的日志
//   - 最多保留 30 个备份文件
//   - 使用本地时间
//   - 不压缩旧文件
func NewWriter(filename ...string) *writer {
	var logFile string
	if len(filename) > 0 {
		logFile = filename[0] // 取第一个文件名
	}

	return &writer{
		_Filename:   logFile,
		_MaxSize:    defaultMaxSize,    // 100MB
		_MaxBackups: defaultMaxBackups, // 保留30个备份
		_MaxAge:     defaultMaxAge,     // 保留30天
		_LocalTime:  true,              // 使用本地时间
		_Compress:   true,              // 默认压缩
	}
}

// SetMaxSize 设置日志文件的最大大小（MB）
// size: 文件大小上限，单位为MB
// 当日志文件达到此大小时会触发轮转
func (w *writer) SetMaxSize(size int) *writer {
	w._MaxSize = size
	return w
}

// SetMaxAge 设置日志文件的最大保留天数
// days: 文件保留天数
// 超过指定天数的日志文件将被删除，设置为0表示不删除
func (w *writer) SetMaxAge(days int) *writer {
	w._MaxAge = days
	return w
}

// SetMaxBackups 设置要保留的最大日志文件数
// count: 要保留的文件数量
// 超过数量限制的旧文件将被删除，设置为0表示不限制数量
func (w *writer) SetMaxBackups(count int) *writer {
	w._MaxBackups = count
	return w
}

// SetLocalTime 设置是否使用本地时间
// local: true表示使用本地时间，false表示使用UTC时间
// 影响日志文件的备份名称中的时间戳
func (w *writer) SetLocalTime(local bool) *writer {
	w._LocalTime = local
	return w
}

// SetCompress 设置是否压缩旧的日志文件
// compress: true表示启用压缩，false表示不压缩
// 启用后，旧的日志文件将被压缩为.gz格式
func (w *writer) SetCompress(compress bool) *writer {
	w._Compress = compress
	return w
}

func (w *writer) Write(p []byte) (n int, err error) {
	if err := w.validate(); err != nil {
		return 0, err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// 清理颜色控制码
	cleanBytes := stripAnsiCodes(p)

	writeLen := int64(len(cleanBytes))
	if writeLen > w.maxBytes() {
		return 0, fmt.Errorf("write length %d exceeds maximum file size %d", writeLen, w.maxBytes())
	}

	if w.file == nil {
		if err = w.openFile(); err != nil {
			return 0, err
		}
	}

	if w.size+writeLen > w.maxBytes() {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = w.file.Write(cleanBytes)
	w.size += int64(n)
	return len(p), err
}

func (w *writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.close()
}

func (w *writer) close() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *writer) rotate() error {
	if err := w.close(); err != nil {
		return fmt.Errorf("failed to close current log file: %v", err)
	}

	currentName := w.filename()
	backupName := w.backupName()
	if err := os.Rename(currentName, backupName); err != nil {
		return fmt.Errorf("failed to backup log file: %v", err)
	}

	if err := w.openFile(); err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	go func() {
		if err := w.processOldFiles(); err != nil {
			// 这里可以考虑添加错误日志记录
			_ = err
		}
	}()

	return nil
}

func (w *writer) openFile() error {
	filename := w.filename()

	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	w.file = f
	w.size = info.Size()
	return nil
}

func (w *writer) filename() string {
	if w._Filename != "" {
		if !filepath.IsAbs(w._Filename) {
			dir, _ := os.Getwd()
			fullPath := filepath.Join(dir, w._Filename)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return filepath.Join(os.TempDir(), filepath.Base(w._Filename))
			}
			return fullPath
		}
		if err := os.MkdirAll(filepath.Dir(w._Filename), 0o755); err != nil {
			return filepath.Join(os.TempDir(), filepath.Base(w._Filename))
		}
		return w._Filename
	}
	name := filepath.Base(os.Args[0]) + "-slog.log"
	return filepath.Join(os.TempDir(), name)
}

func (w *writer) backupName() string {
	dir := filepath.Dir(w.filename())
	filename := filepath.Base(w.filename())
	ext := filepath.Ext(filename)
	prefix := filename[:len(filename)-len(ext)]

	t := time.Now()
	if !w._LocalTime {
		t = t.UTC()
	}

	// 保持原有扩展名，在文件名和扩展名之间插入时间戳
	backupName := fmt.Sprintf("%s-%s%s",
		prefix,                     // 原文件名（不含扩展名）
		t.Format(backupTimeFormat), // 时间戳
		ext,                        // 原扩展名
	)

	return filepath.Join(dir, backupName)
}

func (w *writer) processOldFiles() error {
	files, err := w.oldLogFiles()
	if err != nil {
		return fmt.Errorf("failed to get old log files: %v", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].timestamp.After(files[j].timestamp)
	})

	if w._MaxBackups > 0 && len(files) > w._MaxBackups {
		for _, f := range files[w._MaxBackups:] {
			if err := os.Remove(filepath.Join(filepath.Dir(w.filename()), f.name)); err != nil {
				return fmt.Errorf("failed to remove excess backup file: %v", err)
			}
		}
		files = files[:w._MaxBackups]
	}

	if w._MaxAge > 0 {
		cutoff := time.Now().Add(-time.Duration(w._MaxAge) * 24 * time.Hour)
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				os.Remove(filepath.Join(filepath.Dir(w.filename()), f.name))
			}
		}
	}

	if w._Compress {
		for _, f := range files {
			if !strings.HasSuffix(f.name, compressSuffix) {
				fname := filepath.Join(filepath.Dir(w.filename()), f.name)
				if err := w.compressFile(fname); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (w *writer) compressFile(src string) error {
	const maxRetries = 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = w.try_CompressFile(src)
		if err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
	}
	return fmt.Errorf("failed to compress file after %d retries: %v", maxRetries, err)
}

func (w *writer) try_CompressFile(src string) error {
	dst := src + compressSuffix

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer gzf.Close()

	gz := gzip.NewWriter(gzf)
	defer gz.Close()

	if _, err := io.Copy(gz, f); err != nil {
		os.Remove(dst)
		return err
	}

	return os.Remove(src)
}

func (w *writer) oldLogFiles() ([]logInfo, error) {
	files, err := os.ReadDir(filepath.Dir(w.filename()))
	if err != nil {
		return nil, err
	}

	var logFiles []logInfo
	prefix := filepath.Base(w.filename())
	ext := filepath.Ext(prefix)
	prefix = prefix[:len(prefix)-len(ext)] + "-"

	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasPrefix(name, prefix) {
			if t, err := time.Parse(backupTimeFormat, name[len(prefix):len(name)-len(ext)]); err == nil {
				logFiles = append(logFiles, logInfo{t, name})
			}
		}
	}

	return logFiles, nil
}

func (w *writer) maxBytes() int64 {
	if w._MaxSize == 0 {
		return int64(defaultMaxSize * 1024 * 1024)
	}
	return int64(w._MaxSize) * 1024 * 1024
}

func (w *writer) validate() error {
	if w._MaxSize < 0 {
		return fmt.Errorf("MaxSize cannot be negative")
	}
	if w._MaxAge < 0 {
		return fmt.Errorf("MaxAge cannot be negative")
	}
	if w._MaxBackups < 0 {
		return fmt.Errorf("MaxBackups cannot be negative")
	}
	return nil
}

// stripAnsiCodes 移除ANSI颜色控制码
func stripAnsiCodes(input []byte) []byte {
	if len(input) == 0 {
		return input
	}

	output := make([]byte, 0, len(input))
	for i := 0; i < len(input); i++ {
		if input[i] == '\x1b' && i+1 < len(input) && input[i+1] == '[' {
			// 跳过直到找到 m
			i += 2
			for i < len(input) && input[i] != 'm' {
				i++
			}
		} else {
			output = append(output, input[i])
		}
	}
	return output
}
