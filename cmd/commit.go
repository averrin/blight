// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit [comment]",
	Short: "Make git commit with task url in it",
	Long:  ``,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		Comment := args[0]
		root, _ = filepath.Abs(filepath.Dir(os.Args[0]))

		origin, err := exec.Command("bash", "-c", `git config --get remote.origin.url`).Output()
		if err != nil {
			log.Fatal(err)
		}

		msg := ""

		if strings.Contains(string(origin), "wrke") {

			token = viper.GetString("Token")

			u, err := exec.Command("bash", "-c", `git branch | grep \* | cut -d ' ' -f2 | cut -d '-' -f1`).Output()
			if err != nil {
				log.Fatal(err)
			}
			v2Task := strings.TrimSpace(string(u))
			taskId := viper.GetString(v2Task)
			url := "https://www.wrike.com/open.htm?id=" + v2Task

			if taskId == "" {
				fmt.Println("Converting task id")
				rq := get("ids", map[string]string{
					"type": "ApiV2Task",
					"ids":  fmt.Sprintf("[%v]", v2Task),
				})
				resp := Response{}
				rq.Json(&resp)
				id := IDResponse{}
				mapstructure.Decode(resp.Data[0], &id)
				taskId = id.ID
				viper.Set(v2Task, taskId)
				saveConfig()
			}

			rq := get("tasks/"+taskId, nil)

			resp := Response{}
			rq.Json(&resp)
			task := Task{}
			mapstructure.Decode(resp.Data[0], &task)
			msg = fmt.Sprintf("%s [%s] %s", task.Title, strings.TrimSpace(url), Comment)
		} else {
			msg = Comment
		}
		u, err := exec.Command("git", "commit", `-am`, msg).Output()
		fmt.Println(string(u))
	},
}

var Comment string

func init() {
	RootCmd.AddCommand(commitCmd)
}
