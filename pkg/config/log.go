package config

import (
	"errors"
	"os"

	log "github.com/sirupsen/logrus"
)

type LogTOML struct {
	File  string `mapstructure:"file" toml:"file"  validate:"required"`
	Level string `mapstructure:"level" toml:"level"  validate:"required,oneof=panic fatal error warn info debug trace "`
}

type LogGo struct {
	File  *os.File
	Level log.Level
}

func (lt *LogTOML) Init() (lg *LogGo, err error) {
	lg = &LogGo{}
	fi, err := os.Stat(lt.File)
	if err != nil {
		if !os.IsNotExist(err) {
			log.WithField("log.file", lt.File).Error(err)
			return nil, err
		}
	} else {
		if fi.IsDir() {
			err = errors.New("is directory")
			log.WithField("log.file", lt.File).Error(err)
			return nil, err
		}
	}
	if lt.File == "stdout" {
		lg.File = os.Stdout
	} else if lt.File == "stderr" {
		lg.File = os.Stderr
	} else {
		lg.File, err = os.OpenFile(lt.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			log.WithField("log.file", lt.File).Error(err)
			return nil, err
		}
	}
	lg.Level, err = log.ParseLevel(lt.Level)
	if err != nil {
		log.WithField("log.level", lt.Level).Error()
		return nil, err
	}
	return lg, nil
}
