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
	"os"
	"github.com/sosozhuang/gocomponent/etcd"
)

// etcdCmd represents the etcd command
var etcdCmd = &cobra.Command{
	Use:   "etcd",
	Short: "A simple client for etcd",
	PersistentPreRun: checkEtcd,
}

// etcdCmd represents the create command
var etcdCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create the value of a key",
	Run: etcdCreate,
}

// etcdCmd represents the delete command
var etcdDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key",
	Run: etcdDelete,
}

func init() {
	RootCmd.AddCommand(etcdCmd)
	etcdCmd.Flags().StringArrayVar(&endpoints, "endpoint", []string{}, "machine addresses in the etcd cluster")

	etcdCmd.PersistentFlags().StringVar(&key, "key", "", "the key to create")
	etcdCmd.PersistentFlags().StringVar(&value, "value", "", "the value to create")
	etcdCmd.AddCommand(etcdCreateCmd, etcdDeleteCmd)

}

func checkEtcd(cmd *cobra.Command, args []string) {
	id = checkEnv(ENV_WORKFLOW_ID)
	if len(endpoints) <= 0 {
		log.Fatalln("At least one endpoint required")
	}
	if key == "" {
		log.Fatalln("The flag key is empty")
	}
}
func checkEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("The env[%s] is empty\n", key)
	}
	return v
}

func etcdCreate(cmd *cobra.Command, args []string) {
	etcd.Create(endpoints, id, key, value)
}

func etcdDelete(cmd *cobra.Command, args []string) {
	etcd.Delete(endpoints, id, key)
}