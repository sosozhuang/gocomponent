package model

import (
	"github.com/sosozhuang/component/types"
	"time"
)

const (
	EventTypeComponentStart  types.EventType = "component_start"
	EventTypeComponentResult types.EventType = "component_result"
	EventTypeComponentStop   types.EventType = "component_stop"
)

type Event struct {
	ID           int64           `sql:"primary_key"`
	ExecuteSeqID int64           `sql:"not null;index:idx_event_1"`
	Type         types.EventType `sql:"not null;type:ENUM('component_start', 'component_result', 'component_stop');index:idx_event_1"`
	Content      string          `sql:"null;type:text"`
	CreatedAt    time.Time
}

func (e *Event) TableName() string {
	return "event"
}

func (e *Event) Save() error {
	return db.Save(e).Error
}