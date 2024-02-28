package logx

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

var internalInstance *zap.SugaredLogger

func init() {
	basic, _ := zap.NewDevelopment(zap.AddCaller(), zap.AddCallerSkip(1))
	internalInstance = basic.Sugar()
}

func Logger() *zap.SugaredLogger {
	return internalInstance
}

func Debug(msg string, keysAndValues ...interface{}) {
	internalInstance.Debugw(msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...interface{}) {
	internalInstance.Infow(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...interface{}) {
	internalInstance.Warnw(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...interface{}) {
	internalInstance.Errorw(msg, keysAndValues...)
}

func With(args ...interface{}) *zap.SugaredLogger {
	return internalInstance.With(args...)
}

func Init(isLocal bool) {
	if isLocal {
		return
	}
	basic, _ := zap.NewProduction(zap.AddCaller(), zap.AddCallerSkip(1))
	internalInstance = basic.Sugar()
}

func InitLoggerWithObserver(isLocal bool) *observer.ObservedLogs {
	fac, logs := observer.New(zapcore.InfoLevel)
	log := zap.New(fac)
	internalInstance = log.Sugar()
	return logs
}
