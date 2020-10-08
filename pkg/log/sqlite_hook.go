package log

import (
	"sync"

	"github.com/iyouport-org/relaybaton/pkg/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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
	record := model.NewRecord(entry)
	hook.db.Create(record)
	return nil
}
