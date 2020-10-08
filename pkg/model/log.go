package model

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Log struct {
	gorm.Model
	Level  uint32
	Func   string
	File   string
	Msg    string
	Stack  string
	Fields string
}

func NewRecord(entry *logrus.Entry) *Log {
	data, err := json.Marshal(map[string]interface{}(entry.Data))
	if err != nil {
		data = []byte{}
	}
	var function string
	var file string
	if entry.HasCaller() {
		function = entry.Caller.Function
		file = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
	} else {
		function = ""
		file = ""
	}
	return &Log{
		Model: gorm.Model{
			CreatedAt: entry.Time,
			UpdatedAt: entry.Time,
		},
		Level:  uint32(entry.Level),
		Func:   function,
		File:   file,
		Msg:    entry.Message,
		Stack:  string(debug.Stack()),
		Fields: string(data),
	}
}

func (record Log) TableName() string {
	return "log"
}
