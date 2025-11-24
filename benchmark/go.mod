module benchmark

go 1.24.2

require (
	github.com/darkit/slog v0.0.0
	github.com/sirupsen/logrus v1.9.3
	go.uber.org/zap v1.27.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8 // indirect
)

replace github.com/darkit/slog => ../
