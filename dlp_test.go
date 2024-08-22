package slog

import (
	"testing"
)

func Test_customSanitize(t *testing.T) {
	// 测试不同协议的URL
	urls := []string{
		"rtsp://10.0.0.19/stream",
		"https://user:password@192.168.1.1/api?param=value",
		"rtsp://admin:admin@10.0.0.1/stream",
		"mqtt://user:password@192.168.1.1:1883/topic",
		"mysql://root:password@127.0.0.1:3306/database",
		"redis://:password@192.168.100.1:6379",
		"ftp://user:password@ftp.example.com/path",
		"sftp://user:password@sftp.example.com/path",
	}

	for _, url := range urls {
		t.Log("Original URL: ", url)
		t.Log("Masked URL: ", customSanitize(url))
		t.Log()
	}
}
