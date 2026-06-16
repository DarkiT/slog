package slog

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriterUsesRestrictedFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are platform-specific on Windows")
	}

	path := filepath.Join(t.TempDir(), "logs", "app.log")
	w := NewWriter(path).SetCompress(false)
	if _, err := w.Write([]byte("secret\n")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat log dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != logDirPerm {
		t.Fatalf("log dir perm = %#o, want %#o", got, os.FileMode(logDirPerm))
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat log file: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != logFilePerm {
		t.Fatalf("log file perm = %#o, want %#o", got, os.FileMode(logFilePerm))
	}
}
