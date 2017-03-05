package types

import (
	"database/sql/driver"
	"encoding/json"
	"k8s.io/client-go/pkg/api/v1"
)

type ComponentType string
type EventType string
type ErrCode uint64

type CommonResp struct {
	OK        bool    `json:"ok"`
	ErrorCode ErrCode `json:"error_code"`
	Message   string  `json:"message"`
}

type Env struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EventScript struct {
	ComponentStart  string `json:"component_start"`
	ComponentResult string `json:"component_result"`
	ComponentStop   string `json:"component_stop"`
}

type ExecutionStatus int
const (
	ComponentExecutionStatusAccepted ExecutionStatus = iota
	ComponentExecutionStatusStarted
	ComponentExecutionStatusFinished
	ComponentExecutionStatusStoped
	ComponentExecutionStatusFailed
)

func (status ExecutionStatus) String() string {
	switch status {
	case ComponentExecutionStatusAccepted:
		return "accepted"
	case ComponentExecutionStatusStarted:
		return "started"
	case ComponentExecutionStatusFinished:
		return "finished"
	case ComponentExecutionStatusStoped:
		return "stoped"
	case ComponentExecutionStatusFailed:
		return "failed"
	default:
		return "undefined"
	}
}

type ExecuteComponentMsg struct {
	ExecuteSeqID int64            `json:"execute_seq_id"`
	ComponentID  int64            `json:"component_id"`
	Status       ExecutionStatus  `json:"status"`
	Type         ComponentType    `json:"type"`
	ImageName    string           `json:"image_name"`
	ImageTag     string           `json:"image_tag"`
	Timeout      int              `json:"timeout"`
	KubeMaster   string           `json:"kube_master"`
	KubeSetting  *json.RawMessage `json:"kube_setting"`
	Input        *json.RawMessage `json:"input"`
	Envs         []Env            `json:"envs"`
	NotifyUrl    `json:"notify_url"`
	KubeResp     *json.RawMessage `json:"kube_resp"`
	Detail       string           `json:"detail"`
	Events       []EventMsg       `json:"events"`
}

type NotifyUrl struct {
	StatusChanged   string `json:"status_changed,omitempty"`
	ComponentStart  string `json:"component_start,omitempty"`
	ComponentResult string `json:"component_result,omitempty"`
	ComponentStop   string `json:"component_stop,omitempty"`
}

type KubeSetting struct {
	Pod     *v1.Pod     `json:"pod,omitempty"`
	Service *v1.Service `json:"service,omitempty"`
}

func (e *EventType) Scan(value interface{}) error {
	*e = EventType(value.([]byte))
	return nil
}

func (e EventType) Value() (driver.Value, error) {
	return string(e), nil
}
