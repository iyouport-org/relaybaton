package config

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"os"
)

type logTOML struct {
	File  string `mapstructure:"file"`
	Level string `mapstructure:"level"`
}

type logGo struct {
	File  *os.File
	Level log.Level
}

func (lt *logTOML) Init() (lg *logGo, err error) {
	lg = &logGo{}
	lg.File, err = os.OpenFile(lt.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.WithField("log.file", lt.File).Error(err)
		return nil, err
	}
	switch lt.Level {
	case "panic":
		lg.Level = log.PanicLevel
	case "fatal":
		lg.Level = log.FatalLevel
	case "error":
		lg.Level = log.ErrorLevel
	case "warn":
		lg.Level = log.WarnLevel
	case "info":
		lg.Level = log.InfoLevel
	case "debug":
		lg.Level = log.DebugLevel
	case "trace":
		lg.Level = log.TraceLevel
	default:
		err = errors.New("unknown log level: " + lt.Level)
		log.WithField("log.level", lt.Level).Error()
		return nil, err
	}
	return lg, nil
}
