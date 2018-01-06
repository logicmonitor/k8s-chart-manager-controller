// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"fmt"

	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var format string

// managecmd represents the manage command
var crdCmd = &cobra.Command{
	Use:   "crd",
	Short: "Dump the custom resource definition to JSON or YAML",
	Run: func(cmd *cobra.Command, args []string) {
		c := &client.Client{}
		if format != "json" && format != "yaml" {
			fmt.Printf("Unknown output format %s. Valid formats are \"yaml\" and \"json\"\n", format)
			return
		}
		fmt.Print(c.GetCRDString(format))
	},
}

func init() {
	log.SetLevel(log.DebugLevel)
	crdCmd.Flags().StringVar(&format, "format", "yaml", "CRD output format (\"json\" or \"yaml\")")
	RootCmd.AddCommand(crdCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// managecmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// managecmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
