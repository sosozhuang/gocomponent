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
	Cancelled
)

type WorkflowModel struct {
	Name  string
	Url   string
	Token string

}

type WorkflowLogModel struct {
	
}

type Workflow struct {
	*WorkflowModel
	log *WorkflowLogModel
	executeID int64
	Status int8
	Stages []StageInterface
}

type Executor interface {
	Name() string
	Execute() error
	setID() (int64, error)
}

type xx interface {
	Name() string
	setID() (int64, error)
}

type WorkflowInterface interface {
	Executor
	xx
	next() []*Workflow
	History() error
}

func (workflow *Workflow) next() []*Workflow {
	return []*Workflow{}
}

func (workflow *Workflow) save() error {
	return nil
}

func (workflow *Workflow) Name() string {
	return workflow.WorkflowModel.Name
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
		log.Printf("Workflow[%s] fetch id failed, reason: %s\n",
			workflow.Name(), err)
		workflow.Status = Failed
		workflow.save()
		return err
	}
	workflow.Status = Running
	workflow.save()
	for _, stage := range workflow.Stages {
		log.Printf("Workflow[%s][%d] stage[%s] will start executing\n",
			workflow.Name(), workflow.executeID, stage.Name())
		if err = stage.Execute(); err != nil {
			log.Printf("Workflow[%s][%d] failed, reason: %s\n",
				workflow.Name(), workflow.executeID, err)
			workflow.Status = Failed
			workflow.save()
			return err
		}
	}
	log.Printf("Workflow[%s][%d] finished\n", workflow.Name(), workflow.executeID)
	workflow.Status = Finished
	workflow.save()
	for _, w := range workflow.next() {
		log.Printf("Workflow[%s] will start executing, triggered by workflow[%s][%d]\n",
			w.Name(), workflow.Name(), workflow.executeID)
		go w.Execute()
	}
	return nil
}

type StageModel struct {
	Name    string
	Timeout int64
}

type StageLogModel struct {
	
}

type Stage struct {
	*StageModel
	log *StageLogModel
	executeID int64
	Status int8
	//*Workflow
	wg      sync.WaitGroup
	tokens  []chan struct{}
	Actions []ActionInterface
}

type StageInterface interface {
	Executor
}

func (stage *Stage) save() error {
	return nil
}

func (stage *Stage) Name() string {
	return stage.StageModel.Name
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
	//log.Printf("Workflow[%s] stage[%s] will start executing\n",
	//	stage.Workflow.Name, stage.Name)
	_, err := stage.setID()
	if err != nil {
		log.Printf("Stage[%s] fetch id failed, reason: %s\n", stage.Name(), err)
		stage.Status = Failed
		stage.save()
		return err
	}
	stage.Status = Running
	stage.save()

	done := make(chan struct{})
	actionResp := make(chan *Response)
	go func() {
		stage.wg.Wait()
		close(done)
		close(actionResp)
	}()
	for _, action := range stage.Actions {
		stage.wg.Add(1)
		log.Printf("Stage[%s][%d] action[%s] will start executing\n",
			stage.Name(), stage.executeID, action.Name())
		go action.Execute(done, actionResp)
	}
	if stage.Timeout > 0 {
		tick := time.Tick(time.Duration(stage.Timeout) * time.Second)
		for {
			select {
			case <-done:
				log.Printf("Stage[%s][%d] finished\n",
					stage.Name(), stage.executeID)
				stage.Status = Finished
				stage.save()
				return nil
			case response, ok := <-actionResp:
				if !ok {
					log.Printf("Stage[%s][%d] finished\n",
						stage.Name(), stage.executeID)
					stage.Status = Finished
					stage.save()
					return nil
				}
				if !response.OK {
					go func() {
						for range actionResp {
						}
					}()
					log.Printf("Stage[%s][%d] failed, reason: %s\n",
						stage.Name, stage.executeID, response.Err)
					stage.Status = Failed
					stage.save()
					return fmt.Errorf("Stage[%s][%d] failed, reason: %s",
						stage.Name, stage.executeID, response.Err)
				}
			case <-tick:
				go func() {
					for range actionResp {
					}
				}()
				log.Printf("stage[%s] time out after %d seconds\n",
					stage.Name, stage.Timeout)
				stage.Status = Failed
				stage.save()
				return fmt.Errorf("Stage[%s][%d] time out", stage.Name(), stage.executeID)
			}
		}
	} else {
		for {
			select {
			case <-done:
				log.Printf("Stage[%s][%d] finished\n",
					stage.Name(), stage.executeID)
				stage.Status = Finished
				stage.save()
				return nil
			case response, ok := <-actionResp:
				if !ok {
					log.Printf("Stage[%s][%d] finished\n",
						stage.Name(), stage.executeID)
					stage.Status = Finished
					stage.save()
					return nil
				}
				if !response.OK {
					go func() {
						for range actionResp {
						}
					}()
					log.Printf("Stage[%s][%d] failed, reason: %s\n",
						stage.Name, stage.executeID, response.Err)
					stage.Status = Failed
					stage.save()
					return fmt.Errorf("Stage[%s][%d] failed, reason: %s",
						stage.Name, stage.executeID, response.Err)
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
	*ActionModel
	executeID int64
	Status int8
	Component
	Inputs []*Link
}

type ActionInterface interface {
	Name() string
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

func (action *Action) Name() string {
	return action.ActionModel.Name
}

func (action *Action) setID() (int64, error) {
	if action.executeID <= 0 {
		//TODO: if return error, id is zero, table index is a problem
		if id, err := model.NextValue("action"); err != nil {
			return nil, err
		} else {
			action.executeID = id
		}
	}
	return action.executeID, nil
}

func (action *Action) save() error {
	return nil
}

func (action *Action) Execute(done <-chan struct{}, actionResp chan<- *Response) {
	//log.Printf("Workflow[%s] stage[%s] action[%s][%d] will start executing\n",
	//	action.Workflow.Name, action.Stage.Name, action.Name, action.executeID)
	_, err := action.setID()
	if err != nil {
		log.Printf("Action[%s] fetch id failed, reason: %s\n", action.Name(), err)
		action.Status = Failed
		action.save()
		return err
	}

	defer action.Stage.wg.Done()
	//TODO: add concurrency limit
	select {
	case <-done:
		log.Printf("Stage cancelled, action[%s][%d] return directly\n",
			action.Name(), action.executeID)
		action.Status = Cancelled
		action.save()
		return
	case action.Stage.tokens <- struct{}{}:
	}
	defer func() { <-action.Stage.tokens }()

	action.Status = Running
	action.save()

	raw := []byte(action.Template)
	for _, input := range action.Inputs {
		raw, err = input.MapData(action.EID, raw)
		if err != nil {
			log.Printf("Action[%s][%d] mapped data error, reason: %s\n",
				 action.Name(), action.executeID, err)
		}
	}
	componentResp := make(chan *Response)
	log.Printf("Action[%s][%d] component[%s] will start executing\n",
		action.Name(), action.executeID, action.Component.Name)
	go action.Component.Start(componentResp)
	var response Response
	if action.Timeout > 0 {
		select {
		case response = <-componentResp:
			log.Printf("Action[%s][%d] response: %s\n",
				 action.Name, action.executeID, response)
			actionResp <- &response
		case <-time.After(time.Duration(action.Timeout) * time.Second):
			log.Printf("Action[%s][%d] time out after %d seconds\n",
				action.Name(), action.executeID, action.Timeout)
			response.OK = false
			response.Err = fmt.Errorf("Action[%s][%d] time out", action.Name, action.executeID)
			actionResp <- &response
		}
	} else {
		response = <-componentResp
		log.Printf("Action[%s][%d] response: %s\n",
			 action.Name, action.executeID, response)
		actionResp <- response
	}
	if response.OK {
		action.Status = Finished
	} else {
		action.Status = Failed
	}
	action.save()
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
	//*Workflow
	//*Stage
	//*Action
	ExitCode string
}

type MesosComponent struct {
}

func (component *K8sComponent) Start(componentResp chan<- *Response) {
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



package model

import (
	"github.com/jinzhu/gorm"
)

var db *gorm.DB

type Sequence struct {
	Seqname   string `gorm:"column:seqname;primary_key;type:varchar(50)"`
	Current   int64  `gorm:"column:current;not null;type:bigint;default:1"`
	Increment int64  `gorm:"column:increment;not null;type:int;default:1"`
}

func (seq *Sequence) TableName() string {
	return "sequence"
}

func CurrentValue(seqname string) (int64, error) {
	return rawSql("SELECT currval(?)", seqname)
}

func NextValue(seqname string) (int64, error) {
	return rawSql("SELECT nextval(?)", seqname)
}

func rawSql(sql, seqname string) (int64, error) {
	var value int64
	if err := db.Raw(sql, seqname).
		Row().Scan(&value); err != nil {
		return 0, err
	} else {
		return value, nil
	}
}
