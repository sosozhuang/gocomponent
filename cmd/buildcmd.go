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
	"time"
	"log"
	"os"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build an image",
	PersistentPreRun: checkBuild,
}

var buildFsmCmd = &cobra.Command{
	Use:   "fsm",
	Short: "build an image for fsm",
	Run: build,
}

var buildDsCmd = &cobra.Command{
	Use:   "ds",
	Short: "build an image for dataserver",
	Run: build,
}

var buildCloudfsCmd = &cobra.Command{
	Use:   "cloudfs",
	Short: "build an image for cloudfs",
	Run: build,
}

func init() {
	RootCmd.AddCommand(buildCmd)
	buildCmd.PersistentFlags().StringVar(&host, "host", "localhost", "the docker daemon host")
	buildCmd.PersistentFlags().Uint16Var(&port, "port", 2376, "the docker daemon port")
	buildCmd.PersistentFlags().StringVar(&registry, "registry", "cloudfs-registry.host.huawei.com:5000", "the private docker registry")
	buildCmd.PersistentFlags().StringVar(&repo, "repo", "str", "the docker repository")
	//buildCmd.PersistentFlags().StringVar(&name, "name", "", "the docker image name")
	//buildCmd.PersistentFlags().StringVar(&buildId, "buildid", "9999", "the buildid of workflow")
	buildCmd.PersistentFlags().StringVar(&contextDir, "contextdir", "", "the image files context directory")

	buildCmd.AddCommand(buildFsmCmd, buildDsCmd, buildCloudfsCmd)

}

func checkBuild(cmd *cobra.Command, args []string) {
	buildId = checkEnv(BUILD_ID)
	if contextDir == "" {
		log.Fatalln("Must specify the contextDir")
	}
	if f, _ := os.Stat(contextDir); !f.IsDir() {
		log.Fatalln("ContextDir: %s must be a directory\n", contextDir)
	}
}

func build(cmd *cobra.Command, args []string) {
	name := cmd.Use
	tag := time.Now().Format("20060102") + "-" + buildId
	client, err := docker.Client(host, port)
	if err != nil {
		log.Fatalln(err)
	}
	image := docker.Image{
		registry, repo, name, tag,
	}
	docker.BuildAndPush(client, &image, contextDir)
}