package types

import "time"

type EventMsg struct {
	//Nounce       string
	//Sign         string
	ExecuteSeqID int64     `json:"execute_seq_id"`
	//ExecutorID   string    `json:"executor_id"`
	Type         EventType `json:"type"`
	Content      string    `json:"content"`
	CreateAt     time.Time `json:"create_at"`
}