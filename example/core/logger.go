package core

import (
	"github.com/darkit/slog"
)

var Logger *slog.Logger

func Init() {
	Logger = slog.Default("main", "module")
}
