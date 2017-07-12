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
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v1"

	"github.com/fatih/color"
	"github.com/ideazxy/requests"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// tasksCmd represents the tasks command
var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List tasks in defined folder",
	Long: `List tasks from folder with statuses and branch names`,
	Run: func(cmd *cobra.Command, args []string) {
		tasks()
	},
}

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
}

var token string
var folder int
var folderV3 string

func tasks() {
	root, _ = filepath.Abs(filepath.Dir(os.Args[0]))
	token = viper.GetString("Token")
	folder = viper.GetInt("Folder")
	folderV3 = viper.GetString("FolderV3")

	if token == "" {
		fmt.Println("Define token")
		os.Exit(1)
	}
	if folder == 0 {
		fmt.Println("Define folder")
		os.Exit(1)
	}

	if folderV3 == "" {
		rq := get("ids", map[string]string{
			"type": "ApiV2Folder",
			"ids":  fmt.Sprintf("[%v]", folder),
		})
		resp := Response{}
		rq.Json(&resp)
		id := IDResponse{}
		mapstructure.Decode(resp.Data[0], &id)
		folderV3 = id.ID
		viper.Set("FolderV3", folderV3)
		saveConfig()
	}
	rq := get(fmt.Sprintf("folders/%v/tasks", folderV3), nil)
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
	rq.SetHeader("Authorization", fmt.Sprintf("bearer %v", token))
	for k, v := range args {
		rq.AddParam(k, v)
	}
	rq.AllowRedirects(true)
	resp, _ := rq.Send()
	// fmt.Println(resp.Text())
	return resp
}

func saveConfig() {
	d, _ := yaml.Marshal(viper.AllSettings())
	err := ioutil.WriteFile(viper.ConfigFileUsed(), d, 0644)
	if err != nil {
		fmt.Println(err)
	}
}
