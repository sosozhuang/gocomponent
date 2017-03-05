package handler

import (
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/sosozhuang/component/module"
	"github.com/sosozhuang/component/types"
	"gopkg.in/macaron.v1"
	"net/http"
)

func CreateEvent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp types.CommonResp
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = EventError + EventReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateEvent marshal data error: " + err.Error())
		}
		return
	}

	var req CreateEventReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("CreateEvent unmarshal data error:", err.Error())
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = EventError + EventUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateEvent marshal data error: " + err.Error())
		}
		return
	}

	err = module.ReceiveEvent(req.ExecuteSeqID, req.Type, req.Content)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = EventError + EventGetActionError
		resp.Message = "receive event error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateEvent marshal data error: " + err.Error())
		}
		return
	}

	httpStatus = http.StatusCreated
	resp.OK = true
	resp.Message = "event received"

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("CreateEvent marshal data error: " + err.Error())
	}
	return
}
