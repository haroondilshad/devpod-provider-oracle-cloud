/*
 * Copyright 2023 DevPod Oracle Provider Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"context"

	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/oracle"
	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/options"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise an instance",
	RunE: func(_ *cobra.Command, args []string) error {
		opts, err := options.FromEnv(true)
		if err != nil {
			return err
		}

		configProvider, err := oracle.CreateOCIConfigurationProvider(opts.OCIConfigFile, opts.OCIProfile)
		if err != nil {
			return err
		}

		client, err := oracle.NewOracle(configProvider)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
