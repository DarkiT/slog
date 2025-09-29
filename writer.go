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
	filePath   string // 文件路径
	maxSize    int    // MB为单位
	maxAge     int    // 天数
	maxBackups int    // 最大备份数
	localTime  bool
	compress   bool

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
//   - 压缩旧文件
func NewWriter(filename ...string) *writer {
	var logFile string
	if len(filename) > 0 {
		logFile = filename[0] // 取第一个文件名
	}

	return &writer{
		filePath:   logFile,
		maxSize:    defaultMaxSize,    // 100MB
		maxBackups: defaultMaxBackups, // 保留30个备份
		maxAge:     defaultMaxAge,     // 保留30天
		localTime:  true,              // 使用本地时间
		compress:   true,              // 默认压缩
	}
}

// SetMaxSize 设置日志文件的最大大小（MB）
// size: 文件大小上限，单位为MB
// 当日志文件达到此大小时会触发轮转
func (w *writer) SetMaxSize(size int) *writer {
	w.maxSize = size
	return w
}

// SetMaxAge 设置日志文件的最大保留天数
// days: 文件保留天数
// 超过指定天数的日志文件将被删除，设置为0表示不删除
func (w *writer) SetMaxAge(days int) *writer {
	w.maxAge = days
	return w
}

// SetMaxBackups 设置要保留的最大日志文件数
// count: 要保留的文件数量
// 超过数量限制的旧文件将被删除，设置为0表示不限制数量
func (w *writer) SetMaxBackups(count int) *writer {
	w.maxBackups = count
	return w
}

// SetLocalTime 设置是否使用本地时间
// local: true表示使用本地时间，false表示使用UTC时间
// 影响日志文件的备份名称中的时间戳
func (w *writer) SetLocalTime(local bool) *writer {
	w.localTime = local
	return w
}

// SetCompress 设置是否压缩旧的日志文件
// compress: true表示启用压缩，false表示不压缩
// 启用后，旧的日志文件将被压缩为.gz格式
func (w *writer) SetCompress(compress bool) *writer {
	w.compress = compress
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

	// 先尝试重命名
	if err := os.Rename(currentName, backupName); err != nil {
		// 重命名失败，尝试恢复文件状态
		if reopenErr := w.openFile(); reopenErr != nil {
			return fmt.Errorf("failed to backup log file (%v) and failed to reopen (%v)", err, reopenErr)
		}
		return fmt.Errorf("failed to backup log file: %v", err)
	}

	// 创建新文件
	if err := w.openFile(); err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}

	// 异步处理旧文件
	go func() {
		if err := w.processOldFiles(); err != nil {
			// 将错误输出到标准错误，避免循环日志问题
			fmt.Fprintf(os.Stderr, "[slog-writer] failed to process old files: %v\n", err)
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
	if w.filePath != "" {
		if !filepath.IsAbs(w.filePath) {
			dir, _ := os.Getwd()
			fullPath := filepath.Join(dir, w.filePath)
			if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
				return filepath.Join(os.TempDir(), filepath.Base(w.filePath))
			}
			return fullPath
		}
		if err := os.MkdirAll(filepath.Dir(w.filePath), 0o755); err != nil {
			return filepath.Join(os.TempDir(), filepath.Base(w.filePath))
		}
		return w.filePath
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
	if !w.localTime {
		t = t.UTC()
	}

	// 使用纳秒时间戳确保唯一性
	backupName := fmt.Sprintf("%s-%s.%09d%s",
		prefix,
		t.Format(backupTimeFormat),
		t.Nanosecond(),
		ext,
	)

	// 如果文件已存在，添加序号确保唯一性
	fullPath := filepath.Join(dir, backupName)
	counter := 1
	for {
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			break
		}
		backupName = fmt.Sprintf("%s-%s.%09d.%d%s",
			prefix,
			t.Format(backupTimeFormat),
			t.Nanosecond(),
			counter,
			ext,
		)
		fullPath = filepath.Join(dir, backupName)
		counter++
	}

	return fullPath
}

func (w *writer) processOldFiles() error {
	// 获取文件列表时获取目录路径，避免在处理过程中路径发生变化
	w.mu.Lock()
	currentDir := filepath.Dir(w.filename())
	w.mu.Unlock()

	files, err := w.oldLogFiles()
	if err != nil {
		return fmt.Errorf("failed to get old log files: %v", err)
	}

	// 按时间排序（最新的在前）
	sort.Slice(files, func(i, j int) bool {
		return files[i].timestamp.After(files[j].timestamp)
	})

	var toDelete []string
	var toCompress []string

	// 决定哪些文件需要删除或压缩
	if w.maxBackups > 0 && len(files) > w.maxBackups {
		for _, f := range files[w.maxBackups:] {
			toDelete = append(toDelete, filepath.Join(currentDir, f.name))
		}
		files = files[:w.maxBackups]
	}

	if w.maxAge > 0 {
		cutoff := time.Now().Add(-time.Duration(w.maxAge) * 24 * time.Hour)
		for _, f := range files {
			if f.timestamp.Before(cutoff) {
				toDelete = append(toDelete, filepath.Join(currentDir, f.name))
			}
		}
	}

	if w.compress {
		for _, f := range files {
			if !strings.HasSuffix(f.name, compressSuffix) {
				toCompress = append(toCompress, filepath.Join(currentDir, f.name))
			}
		}
	}

	// 执行删除操作
	for _, filePath := range toDelete {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			// 记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "[slog-writer] failed to remove old log file %s: %v\n", filePath, err)
		}
	}

	// 执行压缩操作
	for _, filePath := range toCompress {
		if err := w.compressFile(filePath); err != nil {
			// 记录错误但继续处理其他文件
			fmt.Fprintf(os.Stderr, "[slog-writer] failed to compress log file %s: %v\n", filePath, err)
		}
	}

	return nil
}

func (w *writer) compressFile(src string) error {
	const maxRetries = 3
	var err error

	for i := 0; i < maxRetries; i++ {
		err = w.tryCompressfile(src)
		if err == nil {
			return nil
		}
		time.Sleep(time.Millisecond * 100 * time.Duration(i+1))
	}
	return fmt.Errorf("failed to compress file after %d retries: %v", maxRetries, err)
}

func (w *writer) tryCompressfile(src string) error {
	dst := src + "." + compressSuffix

	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			// 文件关闭错误通常不会影响主流程，记录到标准错误即可
			fmt.Fprintf(os.Stderr, "[slog-writer] warning: failed to close source file %s: %v\n", src, closeErr)
		}
	}()

	gzf, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := gzf.Close(); closeErr != nil {
			// 目标文件关闭错误，记录警告
			fmt.Fprintf(os.Stderr, "[slog-writer] warning: failed to close compressed file %s: %v\n", dst, closeErr)
		}
	}()

	gz := gzip.NewWriter(gzf)
	defer func() {
		if closeErr := gz.Close(); closeErr != nil {
			// gzip writer关闭错误，记录警告
			fmt.Fprintf(os.Stderr, "[slog-writer] warning: failed to close gzip writer for %s: %v\n", dst, closeErr)
		}
	}()

	if _, err := io.Copy(gz, f); err != nil {
		os.Remove(dst) // 清理失败的压缩文件
		return err
	}

	// 确保压缩数据写入磁盘
	if err := gz.Close(); err != nil {
		os.Remove(dst)
		return err
	}

	if err := gzf.Close(); err != nil {
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
	baseFilename := filepath.Base(w.filename())
	ext := filepath.Ext(baseFilename)
	prefix := baseFilename[:len(baseFilename)-len(ext)] + "-"

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		name := f.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		// 处理普通备份文件和压缩文件
		var timestampPart string
		if strings.HasSuffix(name, ext+"."+compressSuffix) {
			// 压缩文件: app-2024-01-01T12-00-00.log.gz
			timestampPart = name[len(prefix) : len(name)-len(ext)-len(compressSuffix)-1]
		} else if strings.HasSuffix(name, ext) {
			// 普通文件: app-2024-01-01T12-00-00.log
			timestampPart = name[len(prefix) : len(name)-len(ext)]
		} else {
			continue
		}

		// 解析时间戳（可能包含纳秒和序号）
		if t := w.parseTimestamp(timestampPart); !t.IsZero() {
			logFiles = append(logFiles, logInfo{t, name})
		}
	}

	return logFiles, nil
}

func (w *writer) parseTimestamp(timestampPart string) time.Time {
	// 尝试解析新格式 "2006-01-02T15-04-05.123456789" 格式
	if t, err := time.Parse("2006-01-02T15-04-05.000000000", timestampPart); err == nil {
		return t
	}

	// 尝试解析 "2006-01-02T15-04-05.123456789.1" 格式（带序号）
	parts := strings.Split(timestampPart, ".")
	if len(parts) >= 3 {
		baseTime := strings.Join(parts[:2], ".")
		if t, err := time.Parse("2006-01-02T15-04-05.000000000", baseTime); err == nil {
			return t
		}
	}

	// 尝试解析原始格式 "2006-01-02T15-04-05"（向后兼容）
	if t, err := time.Parse(backupTimeFormat, timestampPart); err == nil {
		return t
	}

	return time.Time{}
}

func (w *writer) maxBytes() int64 {
	if w.maxSize == 0 {
		return int64(defaultMaxSize * 1024 * 1024)
	}
	return int64(w.maxSize) * 1024 * 1024
}

func (w *writer) validate() error {
	if w.maxSize < 0 {
		return fmt.Errorf("MaxSize cannot be negative")
	}
	if w.maxAge < 0 {
		return fmt.Errorf("MaxAge cannot be negative")
	}
	if w.maxBackups < 0 {
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
			// 跳过ANSI转义序列
			i += 2
			for i < len(input) {
				ch := input[i]
				i++
				// ANSI序列以字母结束 (A-Z, a-z)
				if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
					break
				}
			}
			i-- // 回退一位，因为外层循环会+1
		} else {
			output = append(output, input[i])
		}
	}
	return output
}
