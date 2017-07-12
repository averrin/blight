// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v1"

	"github.com/fatih/color"
	"github.com/ideazxy/requests"
	"github.com/jinzhu/configor"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "Lust tasks in defined folder",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		tasks()
	},
}

type ConfigStruct struct {
	Token    string `default:""`
	Folder   int    `default:"0"`
	FolderV3 string `default:""`
}

var Config = ConfigStruct{}
var root string

type Response struct {
	Kind string        `json:"kind"`
	Data []interface{} `json:"data"`
}

type IDResponse struct {
	ID      string `json:"id"`
	APIV2ID string `json:"apiV2Id"`
}

type Task struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"accountId"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	Importance  string    `json:"importance"`
	CreatedDate time.Time `json:"createdDate"`
	UpdatedDate time.Time `json:"updatedDate"`
	Dates       struct {
		Type     string `json:"type"`
		Duration int    `json:"duration"`
		Start    string `json:"start"`
		Due      string `json:"due"`
	} `json:"dates"`
	Scope          string `json:"scope"`
	CustomStatusID string `json:"customStatusId"`
	Permalink      string `json:"permalink"`
	Priority       string `json:"priority"`
	V2ID           int
}

type Status struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Standard bool   `json:"standard"`
	Group    string `json:"group"`
	Hidden   bool   `json:"hidden"`
	Color    string `json:"color"`
}

type Workflow struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Standard       bool     `json:"standard"`
	Hidden         bool     `json:"hidden"`
	CustomStatuses []Status `json:"customStatuses"`
}

func init() {
	RootCmd.AddCommand(tasksCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tasksCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tasksCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}

func tasks() {
	root, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	u, _ := user.Current()
	configor.Load(&Config, path.Join(u.HomeDir, ".blight.yml"))
	if Config.Token == "" {
		fmt.Println("Define token")
		os.Exit(1)
	}
	if Config.Folder == 0 {
		fmt.Println("Define folder")
		os.Exit(1)
	}

	if Config.FolderV3 == "" {
		rq := get("ids", map[string]string{
			"type": "ApiV2Folder",
			"ids":  fmt.Sprintf("[%v]", Config.Folder),
		})
		resp := Response{}
		rq.Json(&resp)
		Config.FolderV3 = resp.Data[0].(IDResponse).ID
		saveConfig()
	}
	rq := get(fmt.Sprintf("folders/%v/tasks", Config.FolderV3), nil)
	resp := Response{}
	rq.Json(&resp)
	tasks := resp.Data
	wfs := false
	statuses := map[string]Status{}
	colors := map[string]func(string, ...interface{}) string{
		"Blue":      color.GreenString,
		"Green":     color.GreenString,
		"":          color.WhiteString,
		"Yellow":    color.YellowString,
		"Orange":    color.YellowString,
		"Turquoise": color.CyanString,
		"DarkCyan":  color.CyanString,
		"Indigo":    color.MagentaString,
		"Red":       color.RedString,
	}
	exclude := []*regexp.Regexp{
		regexp.MustCompile(`.*\[Server\].*`),
		regexp.MustCompile(`.*\[Crafting\].*`),
		regexp.MustCompile(`.*\[FS\].*`),
		regexp.MustCompile(`.*\[UI\].*`),
	}
	sort.Slice(tasks, func(i, j int) bool {
		task := NewTask(tasks[i])
		task2 := NewTask(tasks[j])
		return task.CustomStatusID < task2.CustomStatusID
	})

TaskLoop:
	for _, t := range tasks {
		task := NewTask(t)

		if !wfs {
			rq := get(fmt.Sprintf("accounts/%v/workflows", task.AccountID), nil)
			resp := Response{}
			rq.Json(&resp)
			for _, w := range resp.Data {
				workflow := Workflow{}
				mapstructure.Decode(w, &workflow)
				for _, status := range workflow.CustomStatuses {
					statuses[status.ID] = status
				}
			}
			wfs = true
		}

		for _, re := range exclude {
			if re.MatchString(task.Title) {
				continue TaskLoop
			}
		}

		task.V2ID, _ = strconv.Atoi(task.Permalink[len(task.Permalink)-9 : len(task.Permalink)])
		status := statuses[task.CustomStatusID]
		if status.Name == "Completed" {
			continue
		}
		colorF := colors[status.Color]
		re := regexp.MustCompile(`\[[\w ,\.]*\]`)
		branchName := strings.ToLower(strings.Replace(strings.TrimSpace(re.ReplaceAllString(task.Title, "")), " ", "-", -1))
		re = regexp.MustCompile(`[\.,\(\)'"\\\/:;\!\?]+`)
		branchName = re.ReplaceAllString(branchName, "")
		re = regexp.MustCompile(`-{2,}`)
		branchName = re.ReplaceAllString(branchName, "-")
		branchName = strings.Replace(branchName, "_", "-", -1)

		fmt.Printf("\n%v\n%v\nBranch: %v\n",
			colorF(status.Name),
			task.Title,
			color.BlueString(fmt.Sprintf("%v-%v", task.V2ID, branchName)),
		)
	}
}

func NewTask(data interface{}) Task {
	task := Task{}
	mapstructure.Decode(data, &task)
	return task
}

func get(url string, args map[string]string) *requests.HttpResponse {
	rq := requests.Get(fmt.Sprintf("https://www.wrike.com/api/v3/%v", url))
	rq.SetHeader("Authorization", fmt.Sprintf("bearer %v", Config.Token))
	for k, v := range args {
		rq.AddParam(k, v)
	}
	rq.AllowRedirects(true)
	resp, _ := rq.Send()
	// fmt.Println(resp.Text())
	return resp
}

func saveConfig() {
	d, _ := yaml.Marshal(&Config)
	err := ioutil.WriteFile(path.Join(root, "config.yml"), d, 0644)
	if err != nil {
		fmt.Println(err)
	}
}
