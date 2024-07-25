/*
log 包封装了 slog 包，提供了更简单的接口。并且提供了一个全局的 logger(同时包含了使用[slog.TextHandler]和[slog.JSONHandler]的Logger)，可以直接使用。
package log encapsulates the slog package and provides a simpler interface. And provides a global logger (which contains both the Logger using [slog.TextHandler] and [slog.JSONHandler]), which can be used directly.

# Example

	log.SetLevelInfo()
	log.Debugf("hello %s", "world")
	log.Infof("hello %s", "world")
	log.Warnf("hello %s", "world")
	log.Errorf("hello world")
	log.Debug("hello world", "age", 18)
	log.Info("hello world", "age", 18)
	log.Warn("hello world", "age", 18)
	log.Error("hello world", "age", 18)

	l := log.Default()
	l.LogAttrs(context.Background(), log.LevelInfo, "hello world", log.Int("age", 22))
	l.Log(context.Background(), log.LevelInfo, "hello world", "age", 18)
	l.Debugf("hello %s", "world")
	l.Infof("hello %s", "world")
	l.Warnf("hello %s", "world")
	l.Errorf("hello world")
*/
package slog
