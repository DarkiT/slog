module benchmark

go 1.24.0

require (
	github.com/darkit/slog v0.1.2
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/zap v1.27.1
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
)

replace github.com/darkit/slog => ../
