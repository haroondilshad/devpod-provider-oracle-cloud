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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/oracle"
	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an instance",
	RunE:  createOrStartServer,
}

func createOrStartServer(cmd *cobra.Command, args []string) error {
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

	// Get SSH key
	keyDir := filepath.Join(opts.MachineFolder, ".ssh")
	err = os.MkdirAll(keyDir, 0755)
	if err != nil {
		return errors.Wrap(err, "create key dir")
	}

	privateKeyPath := filepath.Join(keyDir, "id_rsa")
	publicKeyPath := filepath.Join(keyDir, "id_rsa.pub")

	var publicKey string
	var privateKey string

	// Check if keys exist
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		// Generate new keys
		log.Default.Infof("Generating new SSH key pair...")
		publicKey, privateKey, err = client.CreateSSHKeyPair()
		if err != nil {
			return errors.Wrap(err, "create ssh key pair")
		}

		// Write keys to disk
		err = ioutil.WriteFile(privateKeyPath, []byte(privateKey), 0600)
		if err != nil {
			return errors.Wrap(err, "write private key")
		}

		err = ioutil.WriteFile(publicKeyPath, []byte(publicKey), 0644)
		if err != nil {
			return errors.Wrap(err, "write public key")
		}
	} else {
		// Read existing keys
		privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
		if err != nil {
			return errors.Wrap(err, "read private key")
		}
		privateKey = string(privateKeyBytes)

		publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
		if err != nil {
			return errors.Wrap(err, "read public key")
		}
		publicKey = string(publicKeyBytes)
	}

	// Create instance
	request, err := o.BuildInstanceOptions(
		ctx,
		opts.MachineID,
		opts.DiskImage,
		opts.DiskSize,
		opts.MachineType,
		opts.Region,
		opts.AvailabilityDomain,
		opts.CompartmentID,
		publicKey,
	)
	if err != nil {
		return errors.Wrap(err, "build instance options")
	}

	// Launch instance
	_, err = o.computeClient.LaunchInstance(ctx, *request)
	if err != nil {
		return errors.Wrap(err, "launch instance")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(createCmd)
}
