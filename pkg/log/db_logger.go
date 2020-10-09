package log

import (
	"context"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type DBLogger struct {
	*log.Logger
}

func (dbLog *DBLogger) LogMode(level logger.LogLevel) logger.Interface {
	switch level {
	case logger.Silent:
		dbLog.SetLevel(log.PanicLevel)
	case logger.Info:
		dbLog.SetLevel(log.InfoLevel)
	case logger.Warn:
		dbLog.SetLevel(log.WarnLevel)
	case logger.Error:
		dbLog.SetLevel(log.ErrorLevel)
	default:
		dbLog.SetLevel(log.DebugLevel)
	}
	return dbLog
}

func (dbLog *DBLogger) Info(ctx context.Context, msg string, objs ...interface{}) {
	m := make(map[string]interface{}, len(objs)+1)
	m["real_file"] = utils.FileWithLineNum()
	for i, b := range objs {
		m[strconv.Itoa(i+1)] = b
	}
	dbLog.Logger.WithContext(ctx).WithFields(m).Info(msg)

}
func (dbLog *DBLogger) Warn(ctx context.Context, msg string, objs ...interface{}) {
	m := make(map[string]interface{}, len(objs)+1)
	m["real_file"] = utils.FileWithLineNum()
	for i, b := range objs {
		m[strconv.Itoa(i+1)] = b
	}
	dbLog.Logger.WithContext(ctx).WithFields(m).Warn(msg)
}
func (dbLog *DBLogger) Error(ctx context.Context, msg string, objs ...interface{}) {
	m := make(map[string]interface{}, len(objs)+1)
	m["real_file"] = utils.FileWithLineNum()
	for i, b := range objs {
		m[strconv.Itoa(i+1)] = b
	}
	dbLog.Logger.WithContext(ctx).WithFields(m).Error(msg)
}
func (dbLog *DBLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	log.WithFields(log.Fields{
		"real_file": utils.FileWithLineNum(),
		"duration":  float64(elapsed.Nanoseconds()) / 1e6,
		"rows":      rows,
		"sql":       sql,
	}).WithContext(ctx).Trace(err)
}
