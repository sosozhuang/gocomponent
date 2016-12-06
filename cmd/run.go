// Copyright © 2016 soso <sosozhuang@163.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/sosozhuang/gocomponent/docker"
	"log"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a container",
	PersistentPreRun: checkRun,
}

var runFsmCmd = &cobra.Command{
	Use:   "fsm",
	Short: "Run fsm container",
	Run: run,
}

var runDsCmd = &cobra.Command{
	Use:   "ds",
	Short: "Run ds container",
	Run: run,
}

var runCloudfsCmd = &cobra.Command{
	Use:   "cloudfs",
	Short: "Run cloudfs container",
	Run: run,
}


func init() {
	RootCmd.AddCommand(runCmd)
	runCmd.AddCommand(runFsmCmd, runDsCmd, runCloudfsCmd)
	runCmd.PersistentFlags().StringVar(&host, "host", "localhost", "the docker daemon host")
	runCmd.PersistentFlags().Uint16Var(&port, "port", 2376, "the docker daemon port")
	runCmd.PersistentFlags().StringVar(&imageName, "image", "", "docker image name")
	runCmd.PersistentFlags().StringArrayVar(&env, "env", []string{}, "docker container environments")

}

func checkRun(cmd *cobra.Command, args []string) {
	buildId = checkEnv(BUILD_ID)
	if imageName == "" {
		log.Fatalln("Must specify the imageName")
	}
}

func run(cmd *cobra.Command, args []string) {
	name := "workflow-" + cmd.Use + "-" + buildId
	client, err := docker.Client(host, port)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = docker.Run(client, name, imageName, env)
	if err != nil {
		log.Fatalln(err)
	}
}
