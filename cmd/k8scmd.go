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
	"log"
	"github.com/sosozhuang/gocomponent/etcd"
	"html/template"
	"strings"
	"bytes"
	"github.com/sosozhuang/gocomponent/k8s"
	"os"
)

var f k8s.RunFunc

// k8sCmd represents the k8s command
var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "A simple client for kubernetes",
	PersistentPreRun: checkK8sCmd,
}

// k8sCreateCmd represents the create command
var k8sCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a resource in kubernetes",
}

var k8sCreateNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Create a namespace",
	PreRun: func(cmd *cobra.Command, args []string) {
			f = k8s.CreateNamespace
	},
	Run: runK8s,
}

var k8sCreatePodCmd = &cobra.Command{
	Use: "pod",
	Short: "Create a Pod",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.CreatePod
	},
	Run: runK8s,
}

var k8sCreateDaemonsetsCmd = &cobra.Command{
	Use:   "daemonsets",
	Short: "Create deamonsets",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.CreateDaemonsets
	},
	Run: runK8s,
}

var k8sCreateDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Create deployment",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.CreateDeployment
	},
	Run: runK8s,
}

// k8sDeleteCmd represents the delete command
var k8sDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a resource in kubernetes",
}

var k8sDeleteNamespaceCmd = &cobra.Command{
	Use:   "namespace",
	Short: "Create a namespace",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.DeleteNamespace
	},
	Run: runK8s,
}

var k8sDeletePodCmd = &cobra.Command{
	Use: "Pod",
	Short: "Delete a Pod",
	PreRun:func(cmd *cobra.Command, args []string) {
		f = k8s.DeletePod
	},
	Run: runK8s,
}

var k8sDeleteDaemonsetsCmd = &cobra.Command{
	Use:   "daemonsets",
	Short: "Create deamonsets",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.DeleteDaemonsets
	},
	Run: runK8s,
}

var k8sDeleteDeploymentCmd = &cobra.Command{
	Use:   "deployment",
	Short: "Create deployment",
	PreRun: func(cmd *cobra.Command, args []string) {
		f = k8s.DeleteDeployment
	},
	Run: runK8s,
}

func init() {
	RootCmd.AddCommand(k8sCmd)
	k8sCmd.PersistentFlags().StringVar(&master, "master", "", "The address:port of the Kubernetes server")
	k8sCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to the kubeconfig file")
	k8sCmd.PersistentFlags().StringArrayVar(&endpoints, "endpoint", []string{}, "machine addresses in the etcd cluster")
	k8sCmd.PersistentFlags().StringVar(&key, "key", "", "The key of etcd")
	k8sCmd.AddCommand(k8sCreateCmd, k8sDeleteCmd)

	k8sCreateCmd.AddCommand(k8sCreateNamespaceCmd, k8sCreatePodCmd, k8sCreateDaemonsetsCmd, k8sCreateDeploymentCmd)

	k8sDeleteCmd.AddCommand(k8sDeleteNamespaceCmd, k8sDeletePodCmd, k8sDeleteDaemonsetsCmd, k8sDeleteDeploymentCmd)
}

func checkK8sCmd(cmd *cobra.Command, args []string) {
	id = checkEnv(ENV_CO_RUN_ID)
	if master == "" {
		log.Fatalln("Must specify the kubernetes master")
	}
	if len(endpoints) <= 0 {
		log.Fatalln("At least one endpoint required")
	}
	if key == "" {
		log.Fatalln("The key flag is empty")
	}
}

func k8sResourceData() ([]byte, error) {
	value := etcd.Get(endpoints, id, key)
	envMap := make(map[string]string)
	for _, item := range os.Environ() {
		slice := strings.Split(item, "=")
		if len(slice) != 2 {
			log.Printf("Failed to get env data[%s]", item)
			continue
		}
		envMap[slice[0]] = slice[1]
	}
	t, err := template.New("resource").Parse(value)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, envMap); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func runK8s(cmd *cobra.Command, args []string) {

	data, err := k8sResourceData()
	if err != nil {
		log.Fatalln(err)
	}
	client, err := k8s.NewClient(master, configPath)
	if err != nil {
		log.Fatalln(err)
	}
	if _, err = client.Run(data, f); err != nil {
		log.Fatalln(err)
	}
}