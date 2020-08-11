package log

import (
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"sync"
)

type SQLiteHook struct {
	mutex sync.Mutex
	db    *gorm.DB
}

func NewSQLiteHook(db *gorm.DB) *SQLiteHook {
	return &SQLiteHook{
		db: db,
	}
}

func (hook *SQLiteHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *SQLiteHook) Fire(entry *logrus.Entry) error {
	record := NewRecord(entry)
	//hook.mutex.Lock()
	//defer hook.mutex.Unlock()
	hook.db.Create(record)
	return nil
}
