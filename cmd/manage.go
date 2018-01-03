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
	"context"

	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/config"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/controller"
	"github.com/logicmonitor/k8s-chart-manager-controller/pkg/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// managecmd represents the manage command
var manageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Start the Chart Manager controller",
	Run: func(cmd *cobra.Command, args []string) {
		// Retrieve the application configuration.
		chartmgrconfig, err := config.New()
		if err != nil {
			log.Fatalf("Failed to get config: %v", err)
		}

		// Instantiate the Chart Manager controller.
		chartmgrcontroller, err := controller.New(chartmgrconfig)
		if err != nil {
			log.Fatalf("Failed to create Chart Manager controller: %v", err)
		}

		// Create the CRD if it does not already exist.
		_, err = chartmgrcontroller.CreateCustomResourceDefinition()
		if err != nil && !apierrors.IsAlreadyExists(err) {
			log.Fatalf("Failed to create CRD: %v", err)
		}

		// Start the Chart Manager controller.
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		go chartmgrcontroller.Run(ctx) // nolint: errcheck

    // Health check.
		http.HandleFunc("/healthz", healthz.HandleFunc)
		log.Fatal(http.ListenAndServe(":8080", nil))
	},
}

func init() {
	log.SetLevel(log.DebugLevel)
	RootCmd.AddCommand(manageCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// managecmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// managecmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
