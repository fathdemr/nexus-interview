package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger

// Init initialises the global logger.
// mode should be "debug" for development (human-readable) or "release" for production (JSON).
func Init(mode string) error {
	var cfg zap.Config

	if mode == "release" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := cfg.Build(zap.AddCallerSkip(0))
	if err != nil {
		return err
	}

	global = logger
	return nil
}

// L returns the global zap logger.
// Panics if Init has not been called — call Init in main before using this.
func L() *zap.Logger {
	if global == nil {
		panic("logger not initialised: call logger.Init before use")
	}
	return global
}

// S returns the global sugared logger for printf-style logging.
func S() *zap.SugaredLogger {
	return L().Sugar()
}

// Sync flushes any buffered log entries. Call on application shutdown.
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}
