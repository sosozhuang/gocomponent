package handler

import (
	"gopkg.in/macaron.v1"
	"net/http"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
	"github.com/sosozhuang/component/module"
	"github.com/sosozhuang/component/types"
)

func CheckImageScript(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp types.CommonResp
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ImageError + ImageReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}

	var req types.CheckImageScriptReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("CheckImageScript unmarshal data error: ", err.Error())
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ImageError + ImageUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}
	err = module.CheckImageScript(req)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ImageError + ImageScriptError
		resp.Message = "check image script error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}
	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "script check passed"
	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("CheckImageScript marshal data error: " + err.Error())
	}
	return
}


func BuildImage(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp types.CommonResp
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ImageError + ImageReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}

	var req types.ImageSetting
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("CheckImageScript unmarshal data error: ", err.Error())
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ImageError + ImageUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}
	_, err = module.BuildImage(req)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ImageError + ImageScriptError
		resp.Message = "check image script error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CheckImageScript marshal data error: " + err.Error())
		}
		return
	}
	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "script check passed"
	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("CheckImageScript marshal data error: " + err.Error())
	}
	return
}