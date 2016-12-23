package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"build/model"
)

type Response struct {
	OK  bool
	Err error
}

func GetWorkflow(name string) WorkflowInterface {
	return &Workflow{
		WorkflowModel: &WorkflowModel{Name: name},
		Stages:        []*Stage{},
	}
}

const (
	Init = iota
	Running
	Finished
	Failed
)

type WorkflowModel struct {
	Name  string
	Url   string
	Token string
	Status int8
}

type Workflow struct {
	*WorkflowModel
	executeID int64
	Stages []StageInterface
}

type Executor interface {
	Execute() error
	setID() (int64, error)
}

type WorkflowInterface interface {
	Executor
	next() []*Workflow
	History() error
}

func (workflow *Workflow) next() []*Workflow {
	return []*Workflow{}
}

func (workflow *Workflow) save() error {
	return nil
}

func (workflow *Workflow) setID() (int64, error) {
	if workflow.executeID <= 0 {
		//TODO: if return error, id is zero, table index is a problem
		if id, err := model.NextValue("workflow"); err != nil {
			return nil, err
		} else {
			workflow.executeID = id
		}
	}
	return workflow.executeID, nil
}

func (workflow *Workflow) Execute() error {
	_, err := workflow.setID()
	if err != nil {
		log.Printf("Workflow[%s] fetch next id failed, reason: %s\n", err)
		workflow.Status = Failed
		workflow.save()
		return err
	}
	workflow.Status = Running
	workflow.save()
	for _, stage := range workflow.Stages {
		if err = stage.Execute(); err != nil {
			log.Printf("Workflow[%s] execution failed, reason: %s\n", err)
			workflow.Status = Failed
			workflow.save()
			return err
		}
	}
	log.Printf("Workflow[%s] execution finished\n", workflow.Name)
	workflow.Status = Finished
	workflow.save()
	for _, w := range workflow.next() {
		log.Printf("Workflow[%s] will start executing, triggered by workflow[%s]\n", w.Name, workflow.Name)
		go w.Execute()
	}
	return nil
}

type StageModel struct {
	Name    string
	Timeout int64
	Status int8
}

type Stage struct {
	*StageModel
	executeID int64
	*Workflow
	wg      sync.WaitGroup
	tokens  []chan struct{}
	Actions []ActionInterface
}

type StageInterface interface {
	Executor
}

func (stage *Stage) setID() (int64, error) {
	if stage.executeID <= 0 {
		//TODO: if return error, id is zero, table index is a problem
		if id, err := model.NextValue("stage"); err != nil {
			return nil, err
		} else {
			stage.executeID = id
		}
	}
	return stage.executeID, nil
}

func (stage *Stage) Execute() error {
	log.Printf("Workflow[%s] stage[%s] will start executing\n",
		stage.Workflow.Name, stage.Name)
	done := make(chan struct{})
	actionResp := make(chan *Response)
	go func() {
		stage.wg.Wait()
		close(done)
		close(actionResp)
	}()
	for _, action := range stage.Actions {
		stage.wg.Add(1)
		go action.Execute(done, actionResp)
	}
	if stage.Timeout > 0 {
		tick := time.Tick(time.Duration(stage.Timeout) * time.Second)
		for {
			select {
			case <-done:
				log.Printf("Workflow[%s] Stage[%s] executed\n",
					stage.Workflow.Name, stage.Name)
				return nil
			case response, ok := <-actionResp:
				if !ok {
					return nil
				}
				if !response.OK {
					go func() {
						for range actionResp {
						}
					}()
					return response.Err
				}
			case <-tick:
				go func() {
					for range actionResp {
					}
				}()
				log.Printf("Workflow[%s] stage[%s] time out after %d seconds\n",
					stage.Workflow.Name, stage.Name, stage.Timeout)
				return fmt.Errorf("Stage[%s] time out", stage.Name)
			}
		}
	} else {
		for {
			select {
			case <-done:
				return nil
			case response, ok := <-actionResp:
				if !ok {
					return nil
				}
				if !response.OK {
					go func() {
						for range actionResp {
						}
					}()
					return response.Err
				}
			}
		}
	}
}

type ActionModel struct {
	Name    string
	Timeout int64
	Template string
}

type Action struct {
	EID int64
	*ActionModel
	*Workflow
	*Stage
	Component
	Inputs []*Link
}

type ActionInterface interface {
	Execute(chan<- struct{}, chan<- *Response)
	Input() (interface{}, error)
	Output() (interface{}, error)
}
func cancelled(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}
func (action *Action) Execute(done <-chan struct{}, actionResp chan<- *Response) {
	log.Printf("Workflow[%s] stage[%s] action[%s] will start executing\n",
		action.Workflow.Name, action.Stage.Name, action.Name)
	defer action.Stage.wg.Done()
	//TODO: add concurrency limit
	select {
	case <-done:
		log.Printf("Workflow[%s] stage[%s] cancelled, action[%s] return directly\n",
			action.Workflow.Name, action.Stage.Name, action.Name)
		return
	case action.Stage.tokens <- struct{}{}:
	}
	defer func() { <-action.Stage.tokens }()

	raw := []byte(action.Template)
	var err error
	for _, input := range action.Inputs {
		raw, err = input.MapData(action.EID, raw)
		if err != nil {
			log.Printf("Workflow[%s] stage[%s] action[%s] mapped data error, reason: %s\n",
				action.Workflow.Name, action.Stage.Name, action.Name, err)
		}
	}
	componentResp := make(chan *Response)
	go action.Component.Start(componentResp)
	if action.Timeout > 0 {
		select {
		case response := <-componentResp:
			log.Printf("Workflow[%s] stage[%s] action[%s] response: %s\n",
				action.Workflow.Name, action.Stage.Name, action.Name, response)
			actionResp <- response
		case <-time.After(time.Duration(action.Timeout) * time.Second):
			log.Printf("Workflow[%s] stage[%s] action[%s] time out after %d seconds\n",
				action.Workflow.Name, action.Stage.Name, action.Timeout)
			actionResp <- &Response{OK: false, Err: fmt.Errorf("Action[%s] time out", action.Name)}
		}
	} else {
		response := <-componentResp
		log.Printf("Workflow[%s] stage[%s] action[%s] response: %s\n",
			action.Workflow.Name, action.Stage.Name, action.Name, response)
		actionResp <- response
	}
}

func (action *Action) GetData(id int64) string {
	return nil
}

type Component interface {
	Start(chan<- Response)
	Stop() error
}

type ComponentModel struct {
	Name       string
	Image      string
	Kubernetes string
	Mesos      string
	Input      string
	Output     string
}

type K8sComponent struct {
	*ComponentModel
	*Workflow
	*Stage
	*Action
	ExitCode string
}

type MesosComponent struct {
}

func (component *K8sComponent) Start(componentResp chan<- *Response) {
	log.Printf("Workflow[%s] stage[%s] action[%s] component[%s] will start executing\n",
		component.Workflow.Name, component.Stage.Name, component.Action.Name, component.Name)
	time.Sleep(10 * time.Second)
	var response *Response
	if component.ExitCode == "0" {
		response.OK = true
	} else {
		response.OK = false
		response.Err = fmt.Errorf("Component[%s] failed, exit code is %s",
			component.Name, component.ExitCode)
	}
	componentResp <- response
}

func (component *K8sComponent) Stop() error {

	return nil
}

type Mapping struct {
	from, to string
}

type Link struct {
	fromAction *Action
	toAction   *Action
	Mappings   []*Mapping
}

func (link *Link) MapData(id int64, raw []byte) ([]byte, error) {
	var mapped []byte
	for _, mapping := range link.Mappings {
		from := strings.Split(strings.TrimLeft(mapping.from, "."), ".")
		fromPath := from[:len(from)-1]
		fromKey := from[len(from)-1]

		var fromMap, subMap map[string]*json.RawMessage
		err := json.Unmarshal([]byte(link.fromAction.GetData(id)), &fromMap)
		if err != nil {
			log.Printf("Unmarshal data from action[%s] error, reason: %s\n",
				link.fromAction.Name, err)
			return nil, err
		}
		for _, key := range fromPath {
			err = json.Unmarshal(*fromMap[key], &subMap)
			if err != nil {
				log.Printf("Unmarshal data [%v] error, key[%s], reason: %s\n",
					fromMap, key, err)
				return nil, err
			}
			fromMap = subMap
		}
		fmt.Printf("Extract data from action[%s], %s\n",
			link.fromAction.Name, string(*fromMap[fromKey]))

		to := strings.Split(strings.TrimLeft(mapping.to, "."), ".")
		toPath := to[:len(to)-1]
		toKey := to[len(to)-1]

		toMap := make(map[string]interface{})
		err = json.Unmarshal(raw, &toMap)
		if err != nil {
			log.Printf("Unmarshal data [%s] error, reason: %s\n",
				string(raw), err)
			break
		}
		for _, key := range toPath {
			m4, ok := (toMap[key]).(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("Cannot convert %T to map[string]interface{}", toMap[key])
			}
			toMap = m4
		}
		var v interface{}
		if err = json.Unmarshal(*fromMap[fromKey], &v); err != nil {
			return nil, fmt.Errorf("Unmarshal data [%v] error, reason: %s\n",
				string(*fromMap[fromKey]), err)
		}
		toMap[toKey] = v
		mapped, err = json.Marshal(toMap)
		if err != nil {
			log.Printf("Marshal data [%v] error, reason: %s\n",
				toMap, err)
			return nil, err
		}
	}
	return mapped, nil
}

func main() {
}
