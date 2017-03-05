package module

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/sosozhuang/component/model"
	"github.com/sosozhuang/component/types"
	"time"
	"net/http"
	"bytes"
	"encoding/json"
)

func ReceiveEvent(executeSeqID int64, eventType types.EventType, content string) error {
	execution, err := model.SelectComponentExecutionForUpdate(executeSeqID)
	if err != nil {
		return errors.New("select component execution for update error: " + err.Error())
	}

	event := &model.Event{
		ExecuteSeqID: executeSeqID,
		Type:         eventType,
		Content:      content,
	}
	err = event.Save()
	if err != nil {
		execution.Rollback()
		return errors.New("save event error: " + err.Error())
	}

	switch eventType {
	case model.EventTypeComponentStart:
		if execution.Status != types.ComponentExecutionStatusAccepted {
			log.Warnf("component execution %d received event type %s, status is %s not accepted\n", executeSeqID, eventType, execution.Status)
			execution.Rollback()
			return errors.New("component execution status is not accepted")
		}
		execution.Status = types.ComponentExecutionStatusStarted
		execution.Detail = execution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" recevied component_start event, status is started.\n"
		err = execution.Save()
		if err != nil {
			log.Errorln("ReceiveEvent save component execution error:", err)

		}
	case model.EventTypeComponentResult:
		if execution.Status != types.ComponentExecutionStatusStarted &&
			execution.Status != types.ComponentExecutionStatusAccepted {
			log.Warnf("component execution %d received event type %s, status is %s not started\n", executeSeqID, eventType, execution.Status)
			execution.Rollback()
			return errors.New("component execution status is not started")
		}
		execution.Status = types.ComponentExecutionStatusFinished
		execution.Detail = execution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" recevied component_finish event, status is finished.\n"
		err = execution.Save()
		if err != nil {
			log.Errorln("ReceiveEvent save component execution error:", err)
		}
	case model.EventTypeComponentStop:
		if execution.Status != types.ComponentExecutionStatusFinished &&
			execution.Status != types.ComponentExecutionStatusStarted &&
			execution.Status != types.ComponentExecutionStatusAccepted {
			log.Warnf("component execution %d received event type %s, status is %s not finished\n", executeSeqID, eventType, execution.Status)
			execution.Rollback()
			return errors.New("component execution status is not finished")
		}
		//execution.Status = model.ComponentExecutionStatusStoped
		execution.Detail = execution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" recevied component_stop event, going to stop execution.\n"
		err = execution.Save()
		if err != nil {
			log.Errorln("ReceiveEvent save component execution error:", err)
		}
		err = StopComponent(executeSeqID)
		if err != nil {
			return errors.New("stop component error: " + err.Error())
		}
	default:
		log.Warnln("RecevieEvent invalid event type:", string(eventType))
	}
	context := &componentExecutionContext{execution.ComponentExecution}
	eventMsg := types.EventMsg{
		ExecuteSeqID: event.ExecuteSeqID,
		Type: eventType,
		Content: event.Content,
		CreateAt: event.CreatedAt,
	}

	if context.GetIsDebug() {
		go func() {
			value, ok := cache.Get(executeSeqID)
			if !ok {
				log.Warnf("Component message channel key %d not exist\n", executeSeqID)
				return
			}

			c, ok := value.(chan types.ExecuteComponentMsg)
			if !ok {
				log.Errorf("Can't convert type %T to message channel\n", value)
				return
			}

			c <- types.ExecuteComponentMsg{
				ExecuteSeqID: executeSeqID,
				Status:       execution.Status,
				Events:       []types.EventMsg{eventMsg},
			}
		}()
	} else {
		var url string
		notifyUrl := context.GetNotifyUrl()
		switch eventType {
		case model.EventTypeComponentStart:
			url = notifyUrl.ComponentStart
		case model.EventTypeComponentResult:
			url = notifyUrl.ComponentResult
		case model.EventTypeComponentStop:
			url = notifyUrl.ComponentStop
		}
		if url != "" {
			go func() {
				var msg types.ExecuteComponentMsg
				msg.ExecuteSeqID = context.GetExecuteSeqID()
				msg.ComponentID = context.GetComponentID()
				msg.Status = context.GetStatus()
				msg.Type = context.GetType()
				msg.ImageName = context.GetImageName()
				msg.ImageTag = context.GetImageTag()
				msg.Timeout = context.GetTimeout()
				msg.KubeMaster = context.GetKubeMaster()
				kubeSetting := json.RawMessage(context.GetKubeSetting())
				msg.KubeSetting = &kubeSetting
				input := json.RawMessage(context.GetInput())
				msg.Input = &input
				msg.Envs = context.GetEnvs()
				msg.NotifyUrl = context.GetNotifyUrl()
				kubeResp := json.RawMessage(context.GetKubeResp())
				msg.KubeResp = &kubeResp
				msg.Detail = context.GetDetail()
				msg.Events = []types.EventMsg{eventMsg}

				body, err := json.Marshal(msg)
				if err != nil {
					log.Errorln("ReceiveEvent marshal eventMsg error:", err.Error())
					return
				}
				resp, err := http.Post(url, "application/json", bytes.NewReader(body))
				if err != nil {
					log.Errorf("ReceiveEvent send event to %s error: %s\n", url, err)
					return
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					log.Errorf("ReceiveEvent send event to %s status code: %d\n", url, resp.StatusCode)
				}
			}()
		} else {
			log.Warnf("ReceiveEvent execute seq id %d, type %v, can't find notify url\n", executeSeqID, eventType)
		}
	}
	return nil
}
