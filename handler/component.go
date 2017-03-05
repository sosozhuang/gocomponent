/*
Copyright 2014 Huawei Technologies Co., Ltd. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"encoding/json"
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/go-macaron/sockets"
	//"github.com/golang/groupcache/lru"
	//"github.com/gorilla/websocket"
	"github.com/sosozhuang/component/model"
	"github.com/sosozhuang/component/module"
	"github.com/sosozhuang/component/types"
	"gopkg.in/macaron.v1"
	"net/http"
	"strconv"
	"time"
	"github.com/gorilla/websocket"
)

func IndexHandler(ctx *macaron.Context) (httpStatus int, result []byte) {
	httpStatus = http.StatusOK
	result = []byte("Component Backend REST API Service")
	return
}

func TestHandler(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp types.CommonResp
	httpStatus = http.StatusOK
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("TestHandler marshal data error: " + err.Error())
		}
		return
	}
	log.Debugf("receive message:", string(body))
	return
}

func ListComponents(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ListComponentsResp
	var fuzzy bool
	name := ctx.QueryTrim("name")
	version := ctx.QueryTrim("version")
	f := ctx.QueryTrim("fuzzy")
	if f == "" {
		fuzzy = false
	} else {
		var err error
		fuzzy, err = strconv.ParseBool(f)
		if err != nil {
			httpStatus = http.StatusBadRequest
			resp.OK = false
			resp.ErrorCode = ComponentError + ComponentReqBodyError
			resp.Message = "parse query param fuzzy error: " + err.Error()

			result, err = json.Marshal(resp)
			if err != nil {
				log.Errorln("ListComponents marshal data error: " + err.Error())
			}
			return
		}
	}
	pageNum := ctx.QueryInt("page_num")
	if pageNum <= 0 {
		pageNum = 5
	}
	versionNum := ctx.QueryInt("version_num")
	if versionNum <= 0 {
		versionNum = 5
	}
	offset := ctx.QueryInt("offset")
	if offset < 0 {
		offset = 0
	}

	components, err := module.GetComponents(name, version, fuzzy, pageNum, versionNum, offset)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentListError
		resp.Message = "list components error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("ListComponents marshal data error: " + err.Error())
		}
		return
	}

	if len(components) == 0 {
		httpStatus = http.StatusNotFound
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentListError
		resp.Message = "components not found"

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("ListComponents marshal data error: " + err.Error())
		}
		return
	}

	for _, component := range components {
		resp.Components = append(resp.Components, ComponentItem{
			ID:      component.ID,
			Name:    component.Name,
			Version: component.Version,
		})
	}

	httpStatus = http.StatusOK
	resp.OK = true

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("ListComponents marshal data error: " + err.Error())
	}
	return
}

func validateCreateImageSetting(imageName, imageTag string, imageSetting types.ImageSetting) error {
	if imageName == "" && imageTag != "" {
		return errors.New("should not specify image tag")
	}

	if imageName != "" {
		if imageSetting.Dockerfile != "" {
			return errors.New("should not specify dockerfile")
		}
		if imageSetting.Name != "" {
			return errors.New("should not specify build image name")
		}
		if imageSetting.Tag != "" {
			return errors.New("should not specify build image tag")
		}
		if imageSetting.Registry != "" {
			return errors.New("should not specify registry")
		}
		if imageSetting.Username != "" {
			return errors.New("should not specify username")
		}
		if imageSetting.Password != "" {
			return errors.New("should not specify password")
		}
		if imageSetting.ComponentStart != "" {
			return errors.New("should not specify componentstart script")
		}
		if imageSetting.ComponentResult != "" {
			return errors.New("should not specify componentresult script")
		}
		if imageSetting.ComponentStop != "" {
			return errors.New("should not specify componentstop script")
		}
	} else {
		if imageSetting.Dockerfile == "" {
			return errors.New("should specify dockerfile")
		}
		if imageSetting.Name == "" {
			return errors.New("should specify build image name")
		}
		if imageSetting.Registry == "" {
			return errors.New("should specify registry")
		}
		if imageSetting.ComponentStart == "" {
			return errors.New("should specify componentstart script")
		}
		if imageSetting.ComponentResult == "" {
			return errors.New("should specify componentresult script")
		}
		if imageSetting.ComponentStop == "" {
			return errors.New("should specify componentstop script")
		}
	}

	return nil
}

func CreateComponent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ComponentResp
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateComponent marshal data error: " + err.Error())
		}
		return
	}

	var req ComponentReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("CreateComponent unmarshal data error:", err.Error())
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateComponent marshal data error: " + err.Error())
		}
		return
	}

	err = validateCreateImageSetting(req.ImageName, req.ImageTag, *req.ImageSetting)
	if err != nil {
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentImageError
		resp.Message = "image setting error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateComponent marshal data error: " + err.Error())
		}
		return
	}

	var component model.Component
	component.Name = req.Name
	component.Version = req.Version
	component.Type = 0
	for index, value := range model.ComponentTypes {
		if string(value) == req.Type {
			component.Type = index
			break
		}
	}

	if req.ImageName == "" {
		imageInfo, err := module.BuildImage(*req.ImageSetting)
		if err != nil {
			httpStatus = http.StatusBadRequest
			resp.OK = false
			resp.ErrorCode = ComponentError + ComponentImageError
			resp.Message = "build image error: " + err.Error()

			result, err = json.Marshal(resp)
			if err != nil {
				log.Errorln("CreateComponent marshal data error: " + err.Error())
			}
			return
		}
		component.ImageName = imageInfo.Name
		component.ImageTag = imageInfo.Tag
	} else {
		component.ImageName = req.ImageName
		component.ImageTag = req.ImageTag
	}
	data, err := json.Marshal(req.ImageSetting)
	if err != nil {
		log.Errorln("CreateComponent marshal ImageSetting data error: " + err.Error())
	}
	component.ImageSetting = string(data)
	component.Timeout = req.Timeout
	component.UseAdvanced = req.UseAdvanced
	kubeSetting := types.KubeSetting{Pod: req.Pod, Service: req.Service}
	data, err = json.Marshal(kubeSetting)
	if err != nil {
		log.Errorln("CreateComponent marshal KubeSetting data error: " + err.Error())
	}
	component.KubeSetting = string(data)
	//data, err = json.Marshal(req.Input)
	//if err != nil {
	//	log.Errorln("Create component marshal Input data error: " + err.Error())
	//}
	component.Input = string(*req.Input)
	//data, err = json.Marshal(req.Output)
	//if err != nil {
	//	log.Errorln("Create component marshal Output data error: " + err.Error())
	//}
	component.Output = string(*req.Output)
	data, err = json.Marshal(req.Env)
	if err != nil {
		log.Errorln("CreateComponent marshal Env data error: " + err.Error())
	}
	component.Envs = string(data)

	if id, err := module.CreateComponent(&component); err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentCreateError
		resp.Message = "create component error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("CreateComponent marshal data error: " + err.Error())
		}
		return
	} else {
		httpStatus = http.StatusCreated
		resp.ComponentReq = &ComponentReq{
			ID:        id,
			ImageName: component.ImageName,
			ImageTag:  component.ImageTag,
		}
		resp.OK = true
		resp.Message = "component created"
	}

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("CreateComponent marshal data error: " + err.Error())
	}
	return
}

func SaveComponentAsNewVersion(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ComponentResp
	componentID := ctx.Params(":component")
	id, err := strconv.ParseInt(componentID, 10, 64)
	if err != nil || id <= 0 {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("SaveComponentAsNewVersion marshal data error: " + err.Error())
		}
		return
	}

	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("SaveComponentAsNewVersion marshal data error: " + err.Error())
		}
		return
	}

	var req ComponentReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("SaveComponentAsNewVersion unmarshal data error:", err.Error())
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("SaveComponentAsNewVersion marshal data error: " + err.Error())
		}
		return
	}

	if id, err := module.SaveComponentAsNewVersion(id, req.Version); err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentCreateError
		resp.Message = "create component error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("SaveComponentAsNewVersion marshal data error: " + err.Error())
		}
		return
	} else {
		httpStatus = http.StatusCreated
		resp.ComponentReq = &ComponentReq{
			ID: id,
		}
		resp.OK = true
		resp.Message = "component created"
	}

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("SaveComponentAsNewVersion marshal data error: " + err.Error())
	}

	return
}

func GetComponent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ComponentResp
	componentID := ctx.Params(":component")
	id, err := strconv.ParseInt(componentID, 10, 64)
	if err != nil || id <= 0 {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponent marshal data error: " + err.Error())
		}
		return
	}
	component, err := module.GetComponentByID(id)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetError
		resp.Message = "get component by id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponent marshal data error: " + err.Error())
		}
		return
	}
	if component == nil {
		httpStatus = http.StatusNotFound
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetError
		resp.Message = "component not found"

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponent marshal data error: " + err.Error())
		}
		return
	}

	httpStatus = http.StatusOK
	resp.OK = true

	resp.ComponentReq = new(ComponentReq)
	resp.ID = component.ID
	resp.Name = component.Name
	resp.Version = component.Version
	resp.ImageName = component.ImageName
	resp.ImageTag = component.ImageTag
	resp.ImageSetting = new(types.ImageSetting)
	if err := json.Unmarshal([]byte(component.ImageSetting), resp.ImageSetting); err != nil {
		log.Errorln("GetComponent unmarshal ImageSetting data error: " + err.Error())
	}
	resp.Timeout = component.Timeout
	resp.Type = string(model.ComponentTypes[component.Type])
	resp.UseAdvanced = component.UseAdvanced
	if err := json.Unmarshal([]byte(component.Envs), &resp.Env); err != nil {
		log.Errorln("GetComponent unmarshal Environment data error: " + err.Error())
	}
	resp.Input = new(json.RawMessage)
	if err := json.Unmarshal([]byte(component.Input), resp.Input); err != nil {
		log.Errorln("GetComponent unmarshal Input data error: " + err.Error())
	}
	resp.Output = new(json.RawMessage)
	if err := json.Unmarshal([]byte(component.Output), resp.Output); err != nil {
		log.Errorln("GetComponent unmarshal Input data error: " + err.Error())
	}

	var kubeSetting types.KubeSetting
	if json.Unmarshal([]byte(component.KubeSetting), &kubeSetting); err != nil {
		log.Errorln("GetComponent unmarshal KubeSetting data error: " + err.Error())
	}
	resp.Pod = kubeSetting.Pod
	resp.Service = kubeSetting.Service

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("GetComponent marshal data error: " + err.Error())
	}
	return
}

func validateUpdateImageSetting(imageName, imageTag string, imageSetting types.ImageSetting, old model.Component) (bool, error) {
	if imageName == "" && imageTag != "" {
		return false, errors.New("should not specify image tag")
	}

	if imageName == "" {
		if imageSetting.Dockerfile == "" {
			return false, errors.New("should specify dockerfile")
		}
		if imageSetting.Name == "" {
			return false, errors.New("should specify build image name")
		}
		if imageSetting.Registry == "" {
			return false, errors.New("should specify registry")
		}
		if imageSetting.ComponentStart == "" {
			return false, errors.New("should specify componentstart script")
		}
		if imageSetting.ComponentResult == "" {
			return false, errors.New("should specify componentresult script")
		}
		if imageSetting.ComponentStop == "" {
			return false, errors.New("should specify componentstop script")
		}
	}

	data, err := json.Marshal(imageSetting)
	if err != nil {
		return false, err
	}
	if imageName == old.ImageName && imageTag == old.ImageTag &&
		old.ImageSetting == string(data) {
		return false, nil
	}
	if imageName == "" {
		return true, nil
	}

	if imageName != "" &&
		old.ImageSetting != string(data) {
		if imageSetting.Dockerfile != "" {
			return false, errors.New("should not specify dockerfile")
		}
		if imageSetting.Name != "" {
			return false, errors.New("should not specify build image name")
		}
		if imageSetting.Tag != "" {
			return false, errors.New("should not specify build image tag")
		}
		if imageSetting.Registry != "" {
			return false, errors.New("should not specify registry")
		}
		if imageSetting.Username != "" {
			return false, errors.New("should not specify username")
		}
		if imageSetting.Password != "" {
			return false, errors.New("should not specify password")
		}
		if imageSetting.ComponentStart != "" {
			return false, errors.New("should not specify componentstart script")
		}
		if imageSetting.ComponentResult != "" {
			return false, errors.New("should not specify componentresult script")
		}
		if imageSetting.ComponentStop != "" {
			return false, errors.New("should not specify componentstop script")
		}
		if imageName != old.ImageName || imageTag != old.ImageTag {
			return false, nil
		}
		return false, nil
	}

	return false, nil
}

func UpdateComponent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ComponentResp
	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	var req ComponentReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("UpdateComponent unmarshal data error:", err.Error())
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	componentID := ctx.Params(":component")
	id, err := strconv.ParseInt(componentID, 10, 64)
	if err != nil || id <= 0 {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	if req.ImageName == "" && req.ImageSetting.Name == "" {
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentImageError
		resp.Message = "should specify build image name or component image name"

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	var component model.Component
	component.ID = id
	component.Name = req.Name
	component.Version = req.Version
	component.Type = 0
	for index, value := range model.ComponentTypes {
		if string(value) == req.Type {
			component.Type = index
			break
		}
	}

	old, err := module.GetComponentByID(id)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetError
		resp.Message = "get component by id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}
	if old == nil {
		httpStatus = http.StatusNotFound
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetError
		resp.Message = "component not found"

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	rebuild, err := validateUpdateImageSetting(req.ImageName, req.ImageTag, *req.ImageSetting, *old)
	if err != nil {
		httpStatus = http.StatusMethodNotAllowed
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentImageError
		resp.Message = "image setting error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}

	if rebuild {
		imageInfo, err := module.BuildImage(*req.ImageSetting)
		if err != nil {
			httpStatus = http.StatusBadRequest
			resp.OK = false
			resp.ErrorCode = ComponentError + ComponentImageError
			resp.Message = "build image error: " + err.Error()

			result, err = json.Marshal(resp)
			if err != nil {
				log.Errorln("UpdateComponent marshal data error: " + err.Error())
			}
			return
		}
		component.ImageName = imageInfo.Name
		component.ImageTag = imageInfo.Tag
	} else {
		component.ImageName = req.ImageName
		component.ImageTag = req.ImageTag
	}

	data, err := json.Marshal(req.ImageSetting)
	if err != nil {
		log.Errorln("UpdateComponent marshal ImageSetting data error: " + err.Error())
	}
	component.ImageSetting = string(data)

	component.Timeout = req.Timeout
	component.UseAdvanced = req.UseAdvanced
	kubeSetting := types.KubeSetting{Pod: req.Pod, Service: req.Service}
	//kubeSetting["pod"] = req.Pod
	//kubeSetting["service"] = req.Service
	data, err = json.Marshal(kubeSetting)
	if err != nil {
		log.Errorln("UpdateComponent marshal KubeSetting data error: " + err.Error())
	}
	component.KubeSetting = string(data)
	//data, err = json.Marshal(req.Input)
	//if err != nil {
	//	log.Errorln("UpdateComponent marshal Input data error: " + err.Error())
	//}
	component.Input = string(*req.Input)
	//data, err = json.Marshal(req.Output)
	//if err != nil {
	//	log.Errorln("UpdateComponent marshal Output data error: " + err.Error())
	//}
	component.Output = string(*req.Output)
	data, err = json.Marshal(req.Env)
	if err != nil {
		log.Errorln("UpdateComponent marshal Env data error: " + err.Error())
	}
	component.Envs = string(data)

	if err := module.UpdateComponent(id, &component); err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentUpdateError
		resp.Message = "update component error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("UpdateComponent marshal data error: " + err.Error())
		}
		return
	}
	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "component updated"

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("UpdateComponent marshal data error: " + err.Error())
	}
	return
}

func DeleteComponent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ComponentResp

	componentID := ctx.Params(":component")
	id, err := strconv.ParseInt(componentID, 10, 64)
	if err != nil || id <= 0 {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("DeleteComponent marshal data error: " + err.Error())
		}
		return
	}

	if err := module.DeleteComponent(id); err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentDeleteError
		resp.Message = "delete component error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("DeleteComponent marshal data error: " + err.Error())
		}
		return
	}

	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "component deleted"

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("DeleteComponent marshal data error: " + err.Error())
	}
	return
}

func DebugComponentJson() macaron.Handler {
	options := sockets.Options{
		SkipLogging:    false,
		MaxMessageSize: 655360,
	}
	return sockets.JSON(DebugComponentMsg{}, &options)
}

func DebugComponent(ctx *macaron.Context,
	receiver <-chan *DebugComponentMsg,
	sender chan<- *DebugComponentMsg,
	done <-chan bool,
	disconnect chan<- int,
	errChan <-chan error) {
	id, err := strconv.ParseInt(ctx.Params(":component"), 10, 64)
	if err != nil {
		sender <- &DebugComponentMsg{
			CommonResp: types.CommonResp{
				OK:        false,
				ErrorCode: ComponentError + ComponentParseIDError,
				Message:   "parse component id error: " + err.Error(),
			},
		}
		return
	}

	var context module.ExecutionContext
	executeChan := make(chan types.ExecuteComponentMsg)
	ticker := time.Tick(10 * 60 * time.Second)
	for {
		select {
		case execute, ok := <-executeChan:
			if !ok {
				log.Errorln("DebugComponent event channel closed")
				break
			}
			log.Debugln("DebugComponent event channel received message")
			if len(execute.Events) == 0 {
				sender <- &DebugComponentMsg{
					DebugSeqID: context.GetExecuteSeqID(),
					Status:     execute.Status,
					CommonResp: types.CommonResp{
						OK: true,
					},
				}
			} else {
				stop := false
				for _, item := range execute.Events {
					sender <- &DebugComponentMsg{
						DebugSeqID: context.GetExecuteSeqID(),
						Status:     execute.Status,
						Event:      &item,
						CommonResp: types.CommonResp{
							OK: true,
						},
					}
					if execute.Status == types.ComponentExecutionStatusStoped {
						stop = true
					}
				}
				if stop {
					return
				}
			}
		case msg := <-receiver:
			//if msg.DebugSeqID > 0 {
			//	cache.Remove(msg.DebugSeqID)
				//executeChan = make(chan types.ExecuteComponentMsg)
			//}
			if msg.KubeMaster == "" {
				sender <- &DebugComponentMsg{
					CommonResp: types.CommonResp{
						OK:        false,
						ErrorCode: ComponentError + ComponentDebugError,
						Message:   "should specify kubernetes api server",
					},
				}
				return
			}
			context, err = module.StartComponent(id, "component-debug", msg.KubeMaster, *msg.Input, msg.Envs, types.NotifyUrl{}, true, msg.DebugSeqID, executeChan)
			if err != nil {
				sender <- &DebugComponentMsg{
					CommonResp: types.CommonResp{
						OK:        false,
						ErrorCode: ComponentError + ComponentDebugError,
						Message:   "debug component error: " + err.Error(),
					},
				}
				return
			}
			//cache.Add(context.GetExecuteSeqID(), executeChan)
			sender <- &DebugComponentMsg{
				DebugSeqID: context.GetExecuteSeqID(),
				Input:      msg.Input,
				Status:     context.GetStatus(),
				CommonResp: types.CommonResp{
					OK: true,
				},
			}
		case <-done:
			log.Debugln("DebugComponent socket closed by client")
			return
		case <-ticker:
			log.Debugln("DebugComponent socket closed by server")
			//context, err = module.GetComponentExecution(context.GetExecuteSeqID(), false)
			//if err != nil {
			//	log.Errorln("DebugComponent get component execution by id error: " + err.Error())
			//} else {
			//	switch context.GetStatus() {
			//	case types.ComponentExecutionStatusFinished:
			//		sender <- &DebugComponentMsg{
			//			DebugSeqID: context.GetExecuteSeqID(),
			//			Status:     context.GetStatus(),
			//			CommonResp: types.CommonResp{
			//				OK:      true,
			//				Message: "component debug finished",
			//			},
			//		}
			//	case types.ComponentExecutionStatusFailed:
			//		sender <- &DebugComponentMsg{
			//			DebugSeqID: context.GetExecuteSeqID(),
			//			Status:     context.GetStatus(),
			//			CommonResp: types.CommonResp{
			//				OK:        false,
			//				ErrorCode: ComponentError + ComponentDebugError,
			//				Message:   "component debug failed",
			//			},
			//		}
			//	}
			//	time.Sleep(time.Second)
			//}
			disconnect <- websocket.CloseNormalClosure
			return
		case err := <-errChan:
			log.Errorf("DebugComponent socket error: %s\n", err)
		}
	}
}

func StartComponent(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ExecuteComponentResp
	componentID := ctx.Params(":component")
	id, err := strconv.ParseInt(componentID, 10, 64)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StartComponent marshal data error: " + err.Error())
		}
		return
	}

	body, err := ctx.Req.Body().Bytes()
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentReqBodyError
		resp.Message = "get requrest body error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StartComponent marshal data error: " + err.Error())
		}
		return
	}

	var req ExecuteComponentReq
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Errorln("StartComponent unmarshal data error:", err.Error())
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentUnmarshalError
		resp.Message = "unmarshal data error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StartComponent marshal data error: " + err.Error())
		}
		return
	}

	//if req.ExecutorID == "" || req.KubeMaster == "" {
	//	httpStatus = http.StatusMethodNotAllowed
	//	resp.OK = false
	//	resp.ErrorCode = ComponentError + ComponentExecuteError
	//	resp.Message = "should specify executor id and kubernetes master"
	//
	//	result, err = json.Marshal(resp)
	//	if err != nil {
	//		log.Errorln("Execute component marshal data error: " + err.Error())
	//	}
	//	return
	//}
	context, err := module.StartComponent(id, req.ExecutorName, req.KubeMaster, *req.Input, req.Envs, req.NotifyUrl, false, 0, nil)
	if err != nil {
		log.Errorln("StartComponent error:", err.Error())
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentExecuteError
		resp.Message = "start component error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StartComponent marshal data error: " + err.Error())
		}
		return
	}
	//err = component.Start()
	//if err != nil {
	//	log.Errorln("ExecuteComponent start component error:", err.Error())
	//	httpStatus = http.StatusBadRequest
	//	resp.OK = false
	//	resp.ErrorCode = ComponentError + ComponentUnmarshalError
	//	resp.Message = "start component error: " + err.Error()
	//
	//	result, err = json.Marshal(resp)
	//	if err != nil {
	//		log.Errorln("Execute component marshal data error: " + err.Error())
	//	}
	//	return
	//}
	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "component execution started"
	resp.ExecuteComponentMsg = new(types.ExecuteComponentMsg)
	resp.ExecuteSeqID = context.GetExecuteSeqID()
	resp.ComponentID = context.GetComponentID()
	resp.Status = context.GetStatus()
	resp.Type = context.GetType()
	resp.ImageName = context.GetImageName()
	resp.ImageTag = context.GetImageTag()
	resp.Timeout = context.GetTimeout()
	resp.KubeMaster = req.KubeMaster
	resp.KubeSetting = req.KubeSetting
	resp.Input = req.Input
	resp.Envs = req.Envs
	resp.NotifyUrl = req.NotifyUrl
	kubeResp := json.RawMessage(context.GetKubeResp())
	resp.KubeResp = &kubeResp
	resp.Detail = context.GetDetail()
	resp.Events = context.GetEvents()

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("StartComponent marshal data error: " + err.Error())
	}
	return
}

func StopComponentExecution(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp types.CommonResp
	executeSeqID := ctx.Params(":execution")
	id, err := strconv.ParseInt(executeSeqID, 10, 64)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StopComponentExecution execution marshal data error: " + err.Error())
		}
		return
	}
	//context, err := module.GetComponentExecution(id, false)
	//if err != nil {
	//	httpStatus = http.StatusBadRequest
	//	resp.OK = false
	//	resp.ErrorCode = ComponentError + ComponentGetExecutionError
	//	resp.Message = "get component execution by id error: " + err.Error()
	//
	//	result, err = json.Marshal(resp)
	//	if err != nil {
	//		log.Errorln("stop component execution marshal data error: " + err.Error())
	//	}
	//	return
	//}
	//if context == nil {
	//	httpStatus = http.StatusNotFound
	//	resp.OK = false
	//	resp.ErrorCode = ComponentError + ComponentGetExecutionError
	//	resp.Message = "component execution not found"
	//
	//	result, err = json.Marshal(resp)
	//	if err != nil {
	//		log.Errorln("Stop component execution marshal data error: " + err.Error())
	//	}
	//	return
	//}
	err = module.StopComponent(id)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentStopExecutionError
		resp.Message = "stop component execution error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("StopComponentExecution marshal data error: " + err.Error())
		}
		return
	}

	httpStatus = http.StatusOK
	resp.OK = true
	resp.Message = "component execution stoped"
	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("StopComponentExecution marshal data error: " + err.Error())
	}
	return
}

func GetComponentExecution(ctx *macaron.Context) (httpStatus int, result []byte) {
	var resp ExecuteComponentResp
	executeSeqID := ctx.Params(":execution")
	id, err := strconv.ParseInt(executeSeqID, 10, 64)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentParseIDError
		resp.Message = "parse component id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponentExecution marshal data error: " + err.Error())
		}
		return
	}

	context, err := module.GetComponentExecution(id, true)
	if err != nil {
		httpStatus = http.StatusBadRequest
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetExecutionError
		resp.Message = "get component execution by id error: " + err.Error()

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponentExecution marshal data error: " + err.Error())
		}
		return
	}
	if context == nil {
		httpStatus = http.StatusNotFound
		resp.OK = false
		resp.ErrorCode = ComponentError + ComponentGetExecutionError
		resp.Message = "component execution not found"

		result, err = json.Marshal(resp)
		if err != nil {
			log.Errorln("GetComponentExecution marshal data error: " + err.Error())
		}
		return
	}

	httpStatus = http.StatusOK
	resp.OK = true
	resp.ExecuteComponentMsg = new(types.ExecuteComponentMsg)
	resp.ExecuteSeqID = context.GetExecuteSeqID()
	resp.ComponentID = context.GetComponentID()
	resp.Status = context.GetStatus()
	resp.Type = context.GetType()
	resp.ImageName = context.GetImageName()
	resp.ImageTag = context.GetImageTag()
	resp.Timeout = context.GetTimeout()
	resp.KubeMaster = context.GetKubeMaster()
	kubeSetting := json.RawMessage(context.GetKubeSetting())
	resp.KubeSetting = &kubeSetting
	input := json.RawMessage(context.GetInput())
	resp.Input = &input
	resp.Envs = context.GetEnvs()
	resp.NotifyUrl = context.GetNotifyUrl()
	kubeResp := json.RawMessage(context.GetKubeResp())
	resp.KubeResp = &kubeResp
	resp.Detail = context.GetDetail()
	resp.Events = context.GetEvents()

	result, err = json.Marshal(resp)
	if err != nil {
		log.Errorln("GetComponentExecution marshal data error: " + err.Error())
	}
	return
}
