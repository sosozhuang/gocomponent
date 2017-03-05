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

package cmd

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	"path"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use: "component",
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func getLogFile(name string, append bool) *os.File {
	if name == "" {
		return os.Stdout
	}
	var f *os.File
	fileInfo, err := os.Stat(name)
	if err == nil {
		if fileInfo.IsDir() {
			name = name + string(os.PathSeparator) + "component.log"
			return getLogFile(name, append)
		} else {
			var flag int
			if append {
				flag = os.O_RDWR | os.O_APPEND
			} else {
				flag = os.O_RDWR | os.O_TRUNC
			}
			f, err = os.OpenFile(name, flag, 0)
		}
	} else if os.IsNotExist(err) {
		d := path.Dir(name)
		_, err = os.Stat(d)
		if os.IsNotExist(err) {
			os.MkdirAll(d, 0755)
		}
		f, err = os.Create(name)
	}
	if err != nil {
		f = os.Stdout
		fmt.Println(err)
	}
	return f
}

func setLogLevel(level string) {
	switch level {
	case "panic":
		log.SetLevel(log.PanicLevel)
	case "fatal":
		log.SetLevel(log.FatalLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.WarnLevel)
	}
}
