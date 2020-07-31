package config

import (
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
	lg.Level, err = log.ParseLevel(lt.Level)
	if err != nil {
		log.WithField("log.level", lt.Level).Error()
		return nil, err
	}
	return lg, nil
}
