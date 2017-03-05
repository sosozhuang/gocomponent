package model

import (
	"time"
)

type Executor struct {
	ID        int64  `sql:primary_key`
	Name      string `sql:"not null;type:varchar(30);unique_index:uix_executor_1"`
	Key       string `sql:"not null;type:varchar(30)"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

func SelectExecutorFromName(name string) (r *Executor, err error) {
	var condition, result Executor
	condition.Name = name
	err = db.Where(condition).First(&result).Error
	r = &result
	return
}

func (e *Executor) Save() error {
	return db.Save(e).Error
}
