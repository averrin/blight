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
	"log"
	"os/exec"

	"github.com/atotto/clipboard"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
)

var Copy bool

// openTaskCmd represents the openTask command
var openTaskCmd = &cobra.Command{
	Use:   "openTask",
	Short: "Open task of this branch in browser",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		u, err := exec.Command("bash", "-c", `git branch | grep \* | cut -d ' ' -f2 | cut -d '-' -f1`).Output()
		if err != nil {
			log.Fatal(err)
		}
		url := "https://www.wrike.com/open.htm?id=" + string(u)
		fmt.Println(url)
		if !Copy {
			open.Run(url)
		} else {
			clipboard.WriteAll(url)
		}
	},
}

func init() {
	RootCmd.AddCommand(openTaskCmd)
	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	openTaskCmd.Flags().BoolVarP(&Copy, "copy", "c", false, "Copy link instead of opening")
}
