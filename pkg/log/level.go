package log

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type levelLogger struct {
	level *int32
	mu    sync.Mutex
	log   *zap.Logger
}

type verbose bool

var l *levelLogger

func NewLoggerWithLevel(level int32, option ...zap.Option) {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevel(),
	)

	if option == nil {
		option = []zap.Option{}
	}
	option = append(option, zap.AddCaller(), zap.AddCallerSkip(1))
	l = &levelLogger{
		level: &level,
		mu:    sync.Mutex{},
		log:   zap.New(core, option...),
	}
}

/*
V for log level, normal usage example
globalLogger default level 3, debug level 10
example level:

	api request  4
	api response 9

	services func 5

	db error  9
	db query  11
	db result 15
*/
func V(level int32) verbose {
	return level < *l.level
}

func (v verbose) Info(format string, args ...interface{}) {
	if v {
		l.log.Info(fmt.Sprintf(format, args...))
	}
}

func Fatal(format string, args ...interface{}) {
	l.log.Fatal(fmt.Sprintf(format, args...))

}
