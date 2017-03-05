package module

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	"github.com/sosozhuang/component/model"
	"github.com/sosozhuang/component/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"time"
	"strconv"
	"github.com/golang/groupcache/lru"
)

var cache *lru.Cache
var ServiceUrl string

func init() {
	//TODO: may want to configure max entries number
	cache = lru.New(50)
	cache.OnEvicted = func(key lru.Key, value interface{}) {
		log.Warnf("Component message channel key %v evicted", key)
		channel, ok := value.(chan types.ExecuteComponentMsg)
		if !ok {
			log.Warnf("Can't convert cache value %T to message channel", value)
			return
		}
		close(channel)
	}
}

type Component interface {
	Start()
	Stop()
	GetExecutionContext() (ExecutionContext, error)
}

type componentExecutionContext struct {
	*model.ComponentExecution
}

type ExecutionContext interface {
	GetExecuteSeqID() int64
	GetExecutorID() int64
	GetExecutorName() string
	GetComponentID() int64
	GetStatus() types.ExecutionStatus
	GetType() types.ComponentType
	GetImageName() string
	GetImageTag() string
	GetTimeout() int
	GetIsDebug() bool
	GetKubeMaster() string
	GetKubeSetting() string
	GetInput() string
	GetEnvs() []types.Env
	GetNotifyUrl() types.NotifyUrl
	GetKubeResp() string
	GetDetail() string
	GetEvents() []types.EventMsg
}

func (context *componentExecutionContext) GetExecuteSeqID() int64 {
	return context.ID
}

func (context *componentExecutionContext) GetExecutorID() int64 {
	return context.ExecutorID
}

func (context *componentExecutionContext) GetExecutorName() string {
	return context.Executor.Name
}

func (context *componentExecutionContext) GetComponentID() int64 {
	return context.ComponentID
}

func (context *componentExecutionContext) GetStatus() types.ExecutionStatus {
	return context.Status
}

func (context *componentExecutionContext) GetType() types.ComponentType {
	if context.Type >= len(model.ComponentTypes) {
		log.Errorf("Context has invalid component type: %d\n", context.Type)
		return "undefined"
	}
	return model.ComponentTypes[context.Type]
}

func (context *componentExecutionContext) GetImageName() string {
	return context.ImageName
}

func (context *componentExecutionContext) GetImageTag() string {
	return context.ImageTag
}

func (context *componentExecutionContext) GetTimeout() int {
	return context.Timeout
}

func (context *componentExecutionContext) GetIsDebug() bool {
	return context.IsDebug
}

func (context *componentExecutionContext) GetKubeMaster() string {
	return context.KubeMaster
}

func (context *componentExecutionContext) GetKubeSetting() string {
	return context.KubeSetting
}

func (context *componentExecutionContext) GetInput() string {
	return context.Input
	//input := make(map[string]interface{})
	//err := json.Unmarshal([]byte(context.Input), &input)
	//if err != nil {
	//	log.Warnln("context GetInput unmarshal error:", err)
	//}
	//return input
}

func (context *componentExecutionContext) GetEnvs() []types.Env {
	envs := make([]types.Env, 0)
	err := json.Unmarshal([]byte(context.Envs), &envs)
	if err != nil {
		log.Warnln("GetEnvs unmarshal error:", err)
	}
	return envs
}

func (context *componentExecutionContext) GetNotifyUrl() types.NotifyUrl {
	var notifyUrl types.NotifyUrl
	err := json.Unmarshal([]byte(context.NotifyUrl), &notifyUrl)
	if err != nil {
		log.Warnln("GetNotifyUrl unmarshal NotifyUrls error:", err)
	}
	return notifyUrl
}

func (context *componentExecutionContext) GetKubeResp() string {
	return context.KubeResp
}

func (context *componentExecutionContext) GetDetail() string {
	return context.Detail
}

func (context *componentExecutionContext) GetEvents() []types.EventMsg {
	events := make([]types.EventMsg, 0)
	for _, event := range context.Events {
		events = append(events, types.EventMsg{
			ExecuteSeqID: context.GetExecuteSeqID(),
			Type:     event.Type,
			Content:  event.Content,
			CreateAt: event.CreatedAt,
		})
	}
	return events
}

type kubeComponent struct {
	SeqID int64
	c     *kubernetes.Clientset
	//mu    sync.Mutex
}

func (component *kubeComponent) String() string {
	return fmt.Sprintf("kubernetes component[%d]", component.SeqID)
}

func (component *kubeComponent) Start() {
	if component.SeqID <= 0 {
		log.Errorln("Start component invalid sequence id:", component.SeqID)
		return
	}

	componentExecution, err := model.SelectComponentExecutionForUpdate(component.SeqID)
	if err != nil {
		log.Errorln("Start component select component execution error:", err)
		return
	}
	if componentExecution.Status != types.ComponentExecutionStatusAccepted {
		componentExecution.Rollback()
		log.Errorln("Start Component status is not accepted")
		return
	}
	log.Infof("%s will start executing", component)
	if componentExecution.Timeout > 0 {
		go func() {
			time.Sleep(time.Duration(componentExecution.Timeout) * time.Second)
			componentExecution, err := model.SelectComponentExecutionForUpdate(component.SeqID)
			if err != nil {
				log.Errorln("Component timeout select component execution error:", err)
				return
			}
			if componentExecution.Status == types.ComponentExecutionStatusFinished ||
				componentExecution.Status == types.ComponentExecutionStatusFailed ||
				componentExecution.Status == types.ComponentExecutionStatusStoped {
				componentExecution.Rollback()
				return
			}
			componentExecution.Detail = componentExecution.Detail +
				time.Now().Format("2006-01-02 15:04:05") +
				" execution is timeout.\n"
			err = componentExecution.Save()
			if err != nil {
				log.Errorln("Component timeout save component execution error:", err)
			}

			go component.Stop()
		}()
	}

	context := &componentExecutionContext{componentExecution.ComponentExecution}
	kubeResp, err := component.create(context)
	if err != nil {
		log.Errorln("Start component send request to kubernetes error:", err)
		componentExecution.Status = types.ComponentExecutionStatusFailed
		componentExecution.Detail = componentExecution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" failed to create kubernetes resource: " + err.Error() + ", status is failed.\n"
		data, err := json.Marshal(kubeResp)
		if err != nil {
			log.Errorln("Start component marshal kubeResp error:", err)
		}
		componentExecution.KubeResp = string(data)
		err = componentExecution.Save()
		if err != nil {
			log.Errorln("Start Component save component execution error:", err)
		}
		go component.delete(context)

		component.notifyExecutor(context)
	} else {
		data, err := json.Marshal(kubeResp)
		if err != nil {
			log.Errorln("Start component marshal kubeResp error:", err)
		}
		componentExecution.KubeResp = string(data)
		componentExecution.Detail = componentExecution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" successfully created kubernetes resource, status is accepted.\n"
		err = componentExecution.Save()
		if err != nil {
			log.Errorln("Start Component save component execution error:", err)
		}
	}
}

func buildKubeClient(kubeMaster string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.BuildConfigFromFlags(kubeMaster, "")
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

//func (component *kubeComponent) buildKubeClient(kubeMaster string) error {
//	component.mu.Lock()
//	defer component.mu.Unlock()
//	if component.c == nil {
//		config, err := clientcmd.BuildConfigFromFlags(kubeMaster, "")
//		if err != nil {
//			return err
//		}
//		component.c, err = kubernetes.NewForConfig(config)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

func (component *kubeComponent) create(context ExecutionContext) (*types.KubeSetting, error) {
	//err := component.buildKubeClient(context.GetKubeMaster())
	//if err != nil {
	//	return nil, errors.New("build kubernetes client error: " + err.Error())
	//}

	kubeSetting := new(types.KubeSetting)
	kubeResp := new(types.KubeSetting)
	err := json.Unmarshal([]byte(context.GetKubeSetting()), kubeSetting)
	if err != nil {
		return kubeResp, errors.New("unmarshal KubeSetting error: " + err.Error())
	}
	seqID := strconv.FormatInt(context.GetExecuteSeqID(), 10)
	if kubeSetting.Service != nil {
		kubeSetting.Service.Name = "co-svc-" + seqID
		kubeSetting.Service.Namespace = context.GetExecutorName()
		kubeSetting.Service.Spec.Selector = make(map[string]string)
		kubeSetting.Service.Spec.Selector["CO_EXECUTE_SEQ_ID"] = seqID
		kubeResp.Service, err = component.c.CoreV1().Services(kubeSetting.Service.Namespace).Create(kubeSetting.Service)
		if err != nil {
			log.Errorf("Create kubernetes service[%v] error: %s", kubeSetting.Service, err)
			return kubeResp, errors.New("start service error: " + err.Error())
		}
	}
	if kubeSetting.Pod != nil {
		kubeSetting.Pod.Name = "co-pod-" + seqID
		kubeSetting.Pod.Namespace = context.GetExecutorName()
		kubeSetting.Pod.Labels = make(map[string]string)
		kubeSetting.Pod.Labels["CO_EXECUTE_SEQ_ID"] = seqID
		kubeSetting.Pod.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		for i, container := range kubeSetting.Pod.Spec.Containers {
			if context.GetImageTag() != "" {
				container.Image = context.GetImageName() + ":" + context.GetImageTag()
			} else {
				container.Image = context.GetImageName()
			}
			container.Name = fmt.Sprintf("%s-%s-%d", "co-container", seqID, i)
			//container.ImagePullPolicy = v1.PullAlways
			container.ImagePullPolicy = v1.PullIfNotPresent
			for _, env := range context.GetEnvs() {
				container.Env = append(container.Env, v1.EnvVar{
					Name: env.Key,
					Value: env.Value,
				})
			}
			container.Env = append(container.Env, v1.EnvVar{
				Name: "CO_EXECUTE_SEQ_ID",
				Value: seqID,
			}, v1.EnvVar{
				Name: "CO_EXECUTE_TIMEOUT",
				Value: strconv.Itoa(context.GetTimeout()),
			}, v1.EnvVar{
				Name: "CO_INPUT",
				Value: context.GetInput(),
			}, v1.EnvVar{
				Name: "CO_EVENT_URL",
				Value: ServiceUrl + "/v2/events",
			})
			kubeSetting.Pod.Spec.Containers[i] = container
		}
		kubeResp.Pod, err = component.c.CoreV1().Pods(kubeSetting.Pod.Namespace).Create(kubeSetting.Pod)
		if err != nil {
			log.Errorf("Create kubernetes pod[%v] error: %s", kubeSetting.Pod, err)
			return kubeResp, errors.New("start pod error: " + err.Error())
		}
	}
	return kubeResp, nil
}

func (component *kubeComponent) notifyExecutor(context ExecutionContext) {
	if context.GetIsDebug() {
		value, ok := cache.Get(context.GetExecuteSeqID())
		if !ok {
			log.Warnf("Component message channel key %d not exist\n", context.GetExecuteSeqID())
			return
		}

		c, ok := value.(chan types.ExecuteComponentMsg)
		if !ok {
			log.Errorf("Can't convert type %T to message channel\n", value)
			return
		}

		log.Debugf("NotifyExecutor send debug message to channel")

		c <- types.ExecuteComponentMsg{
			ExecuteSeqID: context.GetExecuteSeqID(),
			Status:       context.GetStatus(),
		}
	} else {
		notifyUrl := context.GetNotifyUrl()
		url := notifyUrl.StatusChanged
		if url != "" {
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
			msg.Events = context.GetEvents()

			body, err := json.Marshal(msg)
			if err != nil {
				log.Errorln("NotifyExecutor marshal eventMsg error:", err.Error())
				return
			}
			resp, err := http.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				log.Errorf("NotifyExecutor send event to %s error: %s\n", url, err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Errorf("NotifyExecutor send event to %s status code: %d\n", url, resp.StatusCode)
			}
		} else {
			log.Warnf("NotifyExecutor execute seq id %d, can't find notify url\n", context.GetExecuteSeqID())
		}
	}
}

func (component *kubeComponent) Stop() {
	if component.SeqID <= 0 {
		log.Errorln("Stop component invalid sequence id:", component.SeqID)
		return
	}

	componentExecution, err := model.SelectComponentExecutionForUpdate(component.SeqID)
	if err != nil {
		log.Errorln("Stop component select component execution error:", err)
		return
	}
	if componentExecution.Status == types.ComponentExecutionStatusStoped ||
		componentExecution.Status == types.ComponentExecutionStatusFailed {
		componentExecution.Rollback()
		log.Errorln("Stop Component status can't be stoped or failed")
		return
	}
	log.Infof("%s will stop executing", component)

	context := &componentExecutionContext{componentExecution.ComponentExecution}
	err = component.delete(context)
	if err != nil {
		componentExecution.Status = types.ComponentExecutionStatusFailed
		componentExecution.Detail = componentExecution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
				" failed to delete kubernetes resource: " + err.Error() + ", status is failed.\n"
	} else {
		componentExecution.Status = types.ComponentExecutionStatusStoped
		componentExecution.Detail = componentExecution.Detail +
			time.Now().Format("2006-01-02 15:04:05") +
			" successfully deleted kubernetes resource, status is stoped.\n"

	}
	err = componentExecution.Save()
	if err != nil {
		log.Errorln("Stop Component save component execution error:", err)
	}
	component.notifyExecutor(context)
}

func (component *kubeComponent) delete(context ExecutionContext) error {
	//err := component.buildKubeClient(context.GetKubeMaster())
	//if err != nil {
	//	return errors.New("build kube client error: " + err.Error())
	//}
	kubeResp := new(types.KubeSetting)
	err := json.Unmarshal([]byte(context.GetKubeResp()), kubeResp)
	if err != nil {
		return errors.New("unmarshal KubeResp error: " + err.Error())
	}
	var errs error
	if kubeResp.Pod != nil {
		err = component.c.CoreV1().Pods(kubeResp.Pod.Namespace).Delete(kubeResp.Pod.Name, &v1.DeleteOptions{})
		if err != nil {
			errs = errors.New("delete pod error: " + err.Error())
		}
	}
	if kubeResp.Service != nil {
		err = component.c.CoreV1().Services(kubeResp.Service.Namespace).Delete(kubeResp.Service.Name, &v1.DeleteOptions{})
		if err != nil {
			if errs != nil {
				errs = errors.New(errs.Error() + ", delete service error: " + err.Error())
			} else {
				errs = errors.New("delete service error: " + err.Error())
			}
		}
	}
	return errs
}

func (component *kubeComponent) GetExecutionContext() (ExecutionContext, error) {
	componentExecution, err := model.SelectComponentLogWithEvents(component.SeqID, true)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.New("get component execution error: " + err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		return nil, errors.New("component execution not found")
	}
	return &componentExecutionContext{componentExecution}, nil
}

func GetComponents(name, version string, fuzzy bool, pageNum, versionNum, offset int) ([]model.Component, error) {
	if name == "" && fuzzy == true {
		fuzzy = false
	}
	components, err := model.SelectComponents(name, version, fuzzy, pageNum, versionNum, offset)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.New("get components error: " + err.Error())
	}
	return components, nil
}

func CreateComponent(component *model.Component) (int64, error) {
	if component.ID != 0 {
		return 0, fmt.Errorf("should not specify component id: %d", component.ID)
	}
	if component.Name == "" {
		return 0, errors.New("should specify component name")
	}
	if component.Version == "" {
		return 0, errors.New("should specify component version")
	}
	if component.ImageName == "" {
		return 0, errors.New("should specify component image name")
	}
	if component.Timeout < 0 {
		log.Warnln("CreateComponent timeout should ge zero")
		component.Timeout = 0
	}

	condition := &model.Component{
		Name:    component.Name,
		Version: component.Version,
	}
	if result, err := condition.SelectComponent(); err != nil && err != gorm.ErrRecordNotFound {
		log.Errorln("CreateComponent query component error:", err.Error())
		return 0, errors.New("query component error: " + err.Error())
	} else if result.ID > 0 {
		return 0, fmt.Errorf("component exists, id is: %d", result.ID)
	}

	if err := component.Create(); err != nil {
		log.Errorln("CreateComponent query component error:", err.Error())
		return 0, errors.New("create component error: " + err.Error())
	}
	return component.ID, nil
}

func SaveComponentAsNewVersion(id int64, version string) (int64, error) {
	if id <= 0 {
		return 0, errors.New("should specify component id")
	}
	if version == "" {
		return 0, errors.New("should specify component version")
	}

	component, err := model.SelectComponentFromID(id)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorln("SaveComponentAsNewVersion query component error:", err.Error())
		return 0, errors.New("query component error: " + err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		return 0, errors.New("component not found")
	}

	component.ID = 0
	component.Version = version
	if err := component.Create(); err != nil {
		log.Errorln("SaveComponentAsNewVersion save component error:", err.Error())
		return 0, errors.New("create component error: " + err.Error())
	}

	return component.ID, nil
}

func GetComponentByID(id int64) (*model.Component, error) {
	if id <= 0 {
		return nil, errors.New("should specify component id")
	}

	component, err := model.SelectComponentFromID(id)
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Errorln("GetComponentByID query component error:", err.Error())
		return nil, errors.New("query component error: " + err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return component, nil
}

func UpdateComponent(id int64, component *model.Component) error {
	component.ID = id
	//if id != component.ID {
	//	return errors.New("component id in path not equals to the id in body")
	//}
	if component.ImageName == "" {
		return errors.New("should specify component image name")
	}
	if component.Timeout < 0 {
		log.Warnln("UpdateComponent timeout should ge zero")
		component.Timeout = 0
	}

	old, err := model.SelectComponentFromID(id)
	if err != nil {
		log.Errorln("UpdateComponent query component error:", err.Error())
		return errors.New("query component error: " + err.Error())
	}
	if old == nil {
		return errors.New("component not found")
	}

	if component.Name != old.Name {
		return errors.New("component name can't be changed")
	}
	if component.Version != old.Version {
		return errors.New("component version can't be changed")
	}
	component.ID = old.ID
	component.CreatedAt = old.CreatedAt
	if err := component.Save(); err != nil {
		log.Errorln("UpdateComponent save component error:", err.Error())
		return errors.New("save component error: " + err.Error())
	}
	return nil
}

func DeleteComponent(id int64) error {
	if id == 0 {
		return errors.New("should specify component id")
	}

	component, err := model.SelectComponentFromID(id)
	if err != nil {
		log.Errorln("DeleteComponent query component error:", err.Error())
		return errors.New("query component error: " + err.Error())
	}
	if component == nil {
		return errors.New("component does not exist")
	}
	if err := component.Delete(); err != nil {
		log.Errorln("DeleteComponent delete component error:", err.Error())
		return errors.New("delete component error: " + err.Error())
	}
	return nil
}

func DebugComponent(id int64, kubeMaster string, input map[string]interface{}, envs []types.Env) (ExecutionContext, error) {
	if id <= 0 {
		return nil, errors.New("component id should greater than zero")
	}
	if kubeMaster == "" {
		return nil, errors.New("should specify kubeMaster when execute a component")
	}
	u, err := url.Parse(kubeMaster)
	if err != nil {
		return nil, errors.New("parse kubeMaster error: " + err.Error())
	}
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("invalid kubeMaster url scheme: " + u.Scheme)
	}
	component, err := GetComponentByID(id)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, errors.New("component not found")
	}
	if component.Type >= len(model.ComponentTypes) {
		return nil, fmt.Errorf("invalid component type: %d", component.Type)
	}

	//component.Input = input
	//component.Envs = envs
	//actionLog, err := NewMockAction(component, kubeMaster, input)
	//if err != nil {
	//	log.Errorln("DebugComponent mock action error: ", err.Error())
	//	return nil, errors.New("mock action error: " + err.Error())
	//}
	//go actionLog.Start()
	//return actionLog, nil
	return nil, nil
}

func validateUrl(rawurl string) error {
	if rawurl == "" {
		return nil
	}
	u, err := url.Parse(rawurl)
	if err != nil {
		return fmt.Errorf("parse url %s error: %s", rawurl, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid url %s scheme: %s", rawurl, u.Scheme)
	}
	return nil
}

func StartComponent(id int64, executorName, kubeMaster string, input json.RawMessage, envs []types.Env, notifyUrl types.NotifyUrl,
		isDebug bool, debugSeqID int64, executeChan chan types.ExecuteComponentMsg) (ExecutionContext, error) {
	if id <= 0 {
		return nil, errors.New("component id should greater than zero")
	}
	if executorName == "" {
		return nil, errors.New("should specify executor name when execute a component")
	}

	if kubeMaster == "" {
		return nil, errors.New("should specify kubernetes master when execute a component")
	}
	u, err := url.Parse(kubeMaster)
	if err != nil {
		return nil, errors.New("parse kubeMaster error: " + err.Error())
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errors.New("invalid kubeMaster url scheme: " + u.Scheme)
	}
	if err := validateUrl(notifyUrl.StatusChanged); err != nil {
		return nil, err
	}
	if err := validateUrl(notifyUrl.ComponentStart); err != nil {
		return nil, err
	}
	if err := validateUrl(notifyUrl.ComponentResult); err != nil {
		return nil, err
	}
	if err := validateUrl(notifyUrl.ComponentStop); err != nil {
		return nil, err
	}
	component, err := GetComponentByID(id)
	if err != nil {
		return nil, err
	}
	if component == nil {
		return nil, errors.New("component not found")
	}
	if component.Type >= len(model.ComponentTypes) {
		return nil, fmt.Errorf("invalid component type: %d", component.Type)
	}
	if isDebug && debugSeqID > 0 {
		cache.Remove(debugSeqID)
	}
	switch model.ComponentTypes[component.Type] {
	case model.ComponentTypeKubernetes:
		executor, err := model.SelectExecutorFromName(executorName)
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, errors.New("select executor from name error: " + err.Error())
		}
		if err == gorm.ErrRecordNotFound {
			executor = new(model.Executor)
			executor.Name = executorName
			executor.Key = ""
			err = executor.Save()
			if err != nil {
				return nil, errors.New("create new executor error: " + err.Error())
			}
		}

		client, err := buildKubeClient(kubeMaster)
		if err != nil {
			return nil, errors.New("build kubernetes client error: " + err.Error())
		}

		namespace, _ := client.CoreV1().Namespaces().Get(executorName)
		//if err != nil {
		//	return nil, errors.New("get namespace error: " + err.Error())
		//}
		if namespace.Name == "" {
			namespace = new(v1.Namespace)
			namespace.Name = executorName
			_, err = client.CoreV1().Namespaces().Create(namespace)
			if err != nil {
				return nil, errors.New("create namespace error: " + err.Error())
			}
		}

		componentExecution := new(model.ComponentExecution)
		componentExecution.ExecutorID = executor.ID
		componentExecution.ComponentID = id
		componentExecution.Status = types.ComponentExecutionStatusAccepted
		componentExecution.Type = component.Type
		componentExecution.Timeout = component.Timeout
		componentExecution.ImageName = component.ImageName
		componentExecution.ImageTag = component.ImageTag
		componentExecution.IsDebug = isDebug
		componentExecution.KubeMaster = kubeMaster
		componentExecution.KubeSetting = component.KubeSetting
		//data, err := json.Marshal(input)
		//if err != nil {
		//	return nil, errors.New("marshal input error: " + err.Error())
		//}
		componentExecution.Input = string(input)
		data, err := json.Marshal(envs)
		if err != nil {
			return nil, errors.New("marshal envs error: " + err.Error())
		}
		componentExecution.Envs = string(data)
		data, err = json.Marshal(notifyUrl)
		if err != nil {
			return nil, errors.New("marshal notifys error: " + err.Error())
		}
		componentExecution.NotifyUrl = string(data)
		componentExecution.KubeResp = "{}"
		componentExecution.Detail = time.Now().Format("2006-01-02 15:04:05") +
			" successfully created execution, status is accepted.\n"
		if err := componentExecution.Save(); err != nil {
			return nil, errors.New("create component log error: " + err.Error())
		}

		if isDebug {
			cache.Add(componentExecution.ID, executeChan)
		}
		kubeComponent := kubeComponent{
			SeqID: componentExecution.ID,
			c: client,
		}
		go kubeComponent.Start()
		return &componentExecutionContext{componentExecution}, nil
	case model.ComponentTypeMesos, model.ComponentTypeSwarm:
		return nil, errors.New("currently only kubernetes component supported")
	default:
		return nil, errors.New("currently only kubernetes component supported")
	}
}

func StopComponent(id int64) error {
	if id <= 0 {
		return errors.New("execution id should greater than zero")
	}
	componentExecution, err := model.SelectComponentLogWithEvents(id, false)
	if err != nil && err != gorm.ErrRecordNotFound {
		return errors.New("get component execution error: " + err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		return errors.New("component execution not found")
	}
	if componentExecution.Status == types.ComponentExecutionStatusStoped ||
		componentExecution.Status == types.ComponentExecutionStatusFailed {
		return errors.New("status can't be stoped or failed")
	}
	switch model.ComponentTypes[componentExecution.Type] {
	case model.ComponentTypeKubernetes:
		client, err := buildKubeClient(componentExecution.KubeMaster)
		if err != nil {
			return errors.New("build kubernetes client error: " + err.Error())
		}
		kubeComponent := kubeComponent{
			SeqID: componentExecution.ID,
			c: client,
		}
		go kubeComponent.Stop()
		return nil
	case model.ComponentTypeMesos, model.ComponentTypeSwarm:
		return errors.New("currently only kubernetes component supported")
	default:
		return errors.New("currently only kubernetes component supported")
	}
}

func GetComponentExecution(id int64, withEvents bool) (ExecutionContext, error) {
	if id <= 0 {
		return nil, errors.New("execution id should greater than zero")
	}
	componentExecution, err := model.SelectComponentLogWithEvents(id, withEvents)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, errors.New("get component execution error: " + err.Error())
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &componentExecutionContext{componentExecution}, nil
}
