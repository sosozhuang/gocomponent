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

package model

import (
	log "github.com/Sirupsen/logrus"
	"github.com/jinzhu/gorm"
	"github.com/sosozhuang/component/types"
	"time"
)

const (
	ComponentTypeKubernetes types.ComponentType = "Kubernetes"
	ComponentTypeMesos      types.ComponentType = "Mesos"
	ComponentTypeSwarm      types.ComponentType = "Swarm"
)

var ComponentTypes = []types.ComponentType{ComponentTypeKubernetes, ComponentTypeMesos, ComponentTypeSwarm}

type Component struct {
	ID           int64  `sql:"primary_key"`
	Name         string `sql:"not null;type:varchar(100);index:idx_component_1"`
	Version      string `sql:"not null;type:varchar(30);index:idx_component_1"`
	Type         int    `sql:"not null;default:0"` //0-kubernetes 1-mesos 2-swarm
	ImageName    string `sql:"not null;type:varchar(100)"`
	ImageTag     string `sql:"null;type:varchar(30)"`
	ImageSetting string `sql:"null;type:text"`
	Timeout      int    `sql:"null;default:0"`
	UseAdvanced  bool   `sql:"not null;default:false"`
	KubeSetting  string `sql:"null;type:text"`
	Input        string `sql:"null;type:text"`
	Output       string `sql:"null;type:text"`
	Envs         string `sql:"null;type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

func (c *Component) TableName() string {
	return "component"
}

func (component *Component) Create() error {
	return db.Create(component).Error
}

func SelectComponentFromID(id int64) (r *Component, err error) {
	var result Component
	err = db.First(&result, id).Error
	r = &result
	return
}

func (condition *Component) SelectComponent() (component *Component, err error) {
	var result Component
	err = db.Where(condition).First(&result).Error
	component = &result
	return
}

func SelectComponents(name, version string, fuzzy bool, pageNum, versionNum, offset int) (components []Component, err error) {
	var offsetCond, cond string
	values := make([]interface{}, 0)
	cond = " where deleted_at is null "
	if name != "" {
		if fuzzy {
			cond = cond + " and name like ? "
			values = append(values, name+"%")
		} else {
			cond = cond + " and name = ? "
			values = append(values, name)
		}
	}
	if version != "" {
		cond = cond + " and version = ? "
		values = append(values, version)
	}
	var max int
	if name != "" && !fuzzy {
		offsetCond = " where version_num > ? and version_num <= ?"
		max = offset + versionNum
		values = append(values, offset, max)
	} else {
		offsetCond = " where page_num > ? and page_num <= ? and version_num <= ?"
		max = offset + pageNum
		values = append(values, offset, max, versionNum)
	}

	components = make([]Component, 0)
	tx := db.Begin()
	defer tx.Rollback()
	err = tx.Exec("set @page_num = 0").Error
	if err != nil {
		return
	}
	err = tx.Exec("set @version_num = 0").Error
	if err != nil {
		return
	}
	err = tx.Exec("set @name = ''").Error
	if err != nil {
		return
	}
	raw := "select id, name, version " +
		"from (select id, name, version, " +
		"(case when @name != name then @page_num := @page_num + 1 else @page_num end) as page_num, " +
		"(case when @name != name then @version_num := 1 else @version_num := @version_num + 1 end) as version_num, " +
		"@name := name " +
		"from component " +
		cond +
		"order by name, version) t" +
		offsetCond
	log.Debugf("SelectComponents raw sql string: %s", raw)
	err = tx.Unscoped().Raw(raw, values...).Find(&components).Error
	return
}

func (component *Component) Save() error {
	return db.Save(component).Error
}

func (component *Component) Delete() error {
	return db.Delete(component).Error
}

type ComponentExecution struct {
	ID          int64                 `sql:"primary_key"`
	ExecutorID  int64                 `sql:"not null"`
	Executor    Executor
	ComponentID int64                 `sql:"not null"`
	Status      types.ExecutionStatus `sql:"not null"`
	Type        int                   `sql:"not null;default:0"` //0-kubernetes 1-mesos 2-swarm
	ImageName   string                `sql:"not null;type:varchar(100)"`
	ImageTag    string                `sql:"null;type:varchar(30)"`
	Timeout     int                   `sql:"null;default:0"`
	IsDebug     bool                  `sql:"not null;default:false"`
	KubeMaster  string                `sql:"not null"`
	KubeSetting string                `sql:"null;type:text"`
	Input       string                `sql:"null;type:text"`
	Envs        string                `sql:"null;type:text"`
	NotifyUrl   string                `sql:"null;type:text"`
	KubeResp    string                `sql:"null;type:text"`
	Detail      string                `sql:"null;type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Events      []Event  `gorm:"ForeignKey:ExecuteSeqID"`
}

func (c *ComponentExecution) TableName() string {
	return "component_execution"
}

func SelectComponentLogFromID(id int64) (r *ComponentExecution, err error) {
	var result ComponentExecution
	err = db.First(&result, id).Error
	r = &result
	return
}

func SelectComponentLogWithEvents(id int64, withEvents bool) (r *ComponentExecution, err error) {
	var result ComponentExecution
	if withEvents {
		err = db.First(&result, id).Error
	} else {
		err = db.Preload("Events").First(&result, id).Error
	}
	r = &result
	return
}

func (c *ComponentExecution) Save() error {
	return db.Save(c).Error
}

type componentExecutionTx struct {
	tx *gorm.DB
	*ComponentExecution
}

func SelectComponentExecutionForUpdate(id int64) (t *componentExecutionTx, err error) {
	tx := db.Begin()
	var result ComponentExecution
	err = tx.Set("gorm:query_option", "FOR UPDATE").Preload("Executor").First(&result, id).Error
	t = &componentExecutionTx{tx, &result}
	return
}

func (t *componentExecutionTx) Save() (err error) {
	err = t.tx.Save(t.ComponentExecution).Error
	if err != nil {
		t.tx.Rollback()
	} else {
		t.tx.Commit()
	}
	return
}

func (t *componentExecutionTx) Rollback() {
	t.tx.Rollback()
}
