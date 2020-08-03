package config

import (
	log "github.com/sirupsen/logrus"
	"os"
)

type LogTOML struct {
	File  string `mapstructure:"file" toml:"file"  validate:"required,file"`
	Level string `mapstructure:"level" toml:"level"  validate:"required"`
}

type LogGo struct {
	File  *os.File
	Level log.Level
}

func (lt *LogTOML) Init() (lg *LogGo, err error) {
	lg = &LogGo{}
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
