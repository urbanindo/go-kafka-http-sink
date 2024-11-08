package logger

import (
	"os"
	"sync"

	"go.elastic.co/ecszap"
	"go.uber.org/zap"
)

var (
	log        *zap.Logger
	onceLogger sync.Once
	Named      func(string) *zap.Logger
	Debug      func(string, ...zap.Field)
	Info       func(string, ...zap.Field)
	Error      func(string, ...zap.Field)
	Warn       func(string, ...zap.Field)
	DPanic     func(string, ...zap.Field)
	Fatal      func(string, ...zap.Field)
)

func init() {
	onceLogger.Do(func() {
		encoderConfig := ecszap.NewDefaultEncoderConfig()
		core := ecszap.NewCore(encoderConfig, os.Stdout, zap.DebugLevel)
		log = zap.New(core, zap.AddCaller())
		Named = log.Named
		Info = log.Info
		Debug = log.Debug
		Error = log.Error
		Warn = log.Warn
		DPanic = log.DPanic
		Fatal = log.Fatal
	})

	log, _ := zap.NewProduction()
	Log = &LogCustom{
		sugaredlogger: log.Sugar(),
	}
}

// var Log *zap.Logger

// func InitLogger() {
// Log, _ = zap.NewProduction()
// }

var Log *LogCustom

type LogCustom struct {
	sugaredlogger *zap.SugaredLogger
}

type LogInterface interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
}

func NewLogger() LogInterface {
	return Log
}

// SetLogger is the setter for log variable, it should be the only way to assign value to log.
// func SetLogger(sugarLogger *zap.SugaredLogger) {
// 	Log = &LogWithContext{
// 		sugarLogger: sugarLogger,
// 		context:     nil,
// 	}
// }

func (l *LogCustom) Info(args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Info(args)
}

func (l *LogCustom) Infof(format string, args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Infof(format, args)
}

func (l *LogCustom) Error(args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Error(args)
}

func (l *LogCustom) Errorf(format string, args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Errorf(format, args)
}

func (l *LogCustom) Fatal(args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Fatal(args)
}

func (l *LogCustom) Fatalf(format string, args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Fatalf(format, args)
}

func (l *LogCustom) Warnf(format string, args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Warnf(format, args)
}

func (l *LogCustom) Debug(args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Debug(args)
}

func (l *LogCustom) Debugf(format string, args ...interface{}) {
	l.sugaredlogger.WithOptions(zap.AddCallerSkip(1)).Debugf(format, args)
}
