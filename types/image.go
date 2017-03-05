package types

import "encoding/json"

type ImageSetting struct {
	//From            ImageInfo `json:"from"`
	Dockerfile  string `json:"dockerfile"`
	ImageInfo   `json:"build"`
	PushInfo    `json:"push"`
	EventScript `json:"event_script"`
}

type ImageInfo struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type PushInfo struct {
	Registry string `json:"registry"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CheckImageScriptReq struct {
	Script string `json:"script"`
}

type BuildImageReq struct {
	TarStream json.RawMessage `json:"tar_stream"`
	ImageInfo `json:"build"`
	PushInfo  `json:"push"`
}

type BuildImageResp struct {
	*ImageInfo `json:"image"`
	CommonResp `json:"common"`
}
