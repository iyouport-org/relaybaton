package log

import (
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"runtime/debug"
)

type Record struct {
	gorm.Model
	Level  uint32
	Func   string
	File   string
	Msg    string
	Stack  string
	Fields string
}

func NewRecord(entry *logrus.Entry) *Record {
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
	return &Record{
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

func (record Record) TableName() string {
	return "log"
}
