package logx

import "go.uber.org/zap"

type (
	ReqLogger interface {
		Debug(msg string, keysAndValues ...interface{})
		Info(msg string, keysAndValues ...interface{})
		Warn(msg string, keysAndValues ...interface{})
		Error(msg string, keysAndValues ...interface{})
		With(args ...interface{})
	}
	reqLogger struct {
		*zap.SugaredLogger
	}
)

func (rl *reqLogger) Debug(msg string, keysAndValues ...interface{}) {
	rl.Debugw(msg, keysAndValues...)
}

func (rl *reqLogger) Info(msg string, keysAndValues ...interface{}) {
	rl.Infow(msg, keysAndValues...)
}

func (rl *reqLogger) Warn(msg string, keysAndValues ...interface{}) {
	rl.Warnw(msg, keysAndValues...)
}

func (rl *reqLogger) Error(msg string, keysAndValues ...interface{}) {
	rl.Errorw(msg, keysAndValues...)
}

func (rl *reqLogger) With(args ...interface{}) {
	rl.SugaredLogger = rl.SugaredLogger.With(args...)
}

func MustNewReqLogger(xid string) ReqLogger {
	rl := &reqLogger{}
	rl.SugaredLogger = With("xid", xid)
	return rl
}
