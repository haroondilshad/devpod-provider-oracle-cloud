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
	"fmt"

	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/oracle"
	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/options"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Retrieve the status of an instance",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := options.FromEnv(false)
		if err != nil {
			return err
		}

		ctx := context.Background()
		
		configProvider, err := oracle.CreateOCIConfigurationProvider(opts.OCIConfigFile, opts.OCIProfile)
		if err != nil {
			return err
		}

		o, err := oracle.NewOracle(configProvider)
		if err != nil {
			return err
		}

		status, err := o.GetInstanceStatus(ctx, opts.MachineID)
		if err != nil {
			return errors.Wrap(err, "get instance status")
		}

		fmt.Print(status)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
