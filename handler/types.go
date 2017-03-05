package handler

import (
	"encoding/json"
	"github.com/sosozhuang/component/types"
	"k8s.io/client-go/pkg/api/v1"
)

type RegisterResp struct {
	InvokerID string
	Key       string
}

type ComponentResp struct {
	*ComponentReq    `json:"component,omitempty"`
	types.CommonResp `json:"common"`
}

type ComponentItem struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ListComponentsResp struct {
	Components       []ComponentItem `json:"components,omitempty"`
	types.CommonResp `json:"common"`
}

type ComponentReq struct {
	ID                  int64            `json:"id"`
	Name                string           `json:"name"`
	Version             string           `json:"version"`
	Input               *json.RawMessage `json:"input,omitempty"`
	Output              *json.RawMessage `json:"output,omitempty"`
	Env                 []types.Env      `json:"env"`
	ImageName           string           `json:"image_name"`
	ImageTag            string           `json:"image_tag"`
	*types.ImageSetting `json:"image_setting,omitempty"`
	Timeout             int         `json:"timeout"`
	Type                string      `json:"type"`
	UseAdvanced         bool        `json:"use_advanced"`
	Pod                 *v1.Pod     `json:"pod,omitempty"`
	Service             *v1.Service `json:"service,omitempty"`
}

type DebugComponentMsg struct {
	DebugSeqID int64                 `json:"debug_seq_id"`
	KubeMaster string                `json:"kube_master"`
	Input      *json.RawMessage      `json:"input,omitempty"`
	Envs       []types.Env           `json:"envs,omitempty"`
	Status     types.ExecutionStatus `json:"status"`
	Event      *types.EventMsg       `json:"event,omitempty"`
	types.CommonResp `json:"common"`
}

type ExecuteComponentReq struct {
	//Nounce      string
	//Sign        string
	ExecutorName    string           `json:"executor_name"`
	KubeMaster      string           `json:"kube_master"`
	KubeSetting     *json.RawMessage `json:"kube_setting"`
	Input           *json.RawMessage `json:"input"`
	Envs            []types.Env      `json:"envs"`
	types.NotifyUrl `json:"notify_url"`
}

type ExecuteComponentResp struct {
	*types.ExecuteComponentMsg `json:"execute,omitempty"`
	types.CommonResp           `json:"common"`
}

type CreateEventReq struct {
	ExecuteSeqID int64           `json:"execute_seq_id"`
	Type         types.EventType `json:"type"`
	Content      string          `json:"content"`
}
