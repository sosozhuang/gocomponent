package model

import (
	log "github.com/Sirupsen/logrus"
	"github.com/containerops/configure"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"os"
)

var db *gorm.DB

func init() {
	var err error
	driver := configure.GetString("database.driver")
	uri := configure.GetString("database.uri")
	if db, err = gorm.Open(driver, uri); err != nil {
		log.Fatalf("Open database connection[%s][%s] error: %s\n", driver, uri, err.Error())
		os.Exit(1)
	}
	db.SetLogger(log.StandardLogger())
	db.DB().Ping()
	db.DB().SetMaxIdleConns(10)
	db.DB().SetMaxOpenConns(100)
	db.SingularTable(true)
}

func CloseDB() {
	if db != nil {
		err := db.Close()
		if err != nil {
			log.Errorln("Close db error:" + err.Error())
		}
	}
}

func Migrate() {
	db.AutoMigrate(&Component{}, &ComponentExecution{}, &Event{}, &Executor{})

	log.Infoln("Component database structs migrated.")
}
