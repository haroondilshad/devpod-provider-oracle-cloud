/*
 * Copyright 2023 Simon Emms <simon@simonemms.com>
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

package options

import (
	"fmt"
	"os"
	"strings"
)

type Options struct {
	MachineID     string
	MachineFolder string

	Region            string
	CompartmentID     string
	AvailabilityDomain string
	DiskImage         string
	DiskSize          string
	MachineType       string
	OCIConfigFile     string
	OCIProfile        string
}

func FromEnv(skipMachine bool) (*Options, error) {
	retOptions := &Options{}

	var err error
	if !skipMachine {
		retOptions.MachineID, err = fromEnvOrError("MACHINE_ID")
		if err != nil {
			return nil, err
		}

		retOptions.MachineFolder, err = fromEnvOrError("MACHINE_FOLDER")
		if err != nil {
			return nil, err
		}
	}

	retOptions.OCIConfigFile = os.Getenv("OCI_CONFIG_FILE")
	if retOptions.OCIConfigFile == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			retOptions.OCIConfigFile = homeDir + "/.oci/config"
		}
	}

	retOptions.OCIProfile = os.Getenv("OCI_PROFILE")
	if retOptions.OCIProfile == "" {
		retOptions.OCIProfile = "DEFAULT"
	}

	retOptions.CompartmentID, err = fromEnvOrError("COMPARTMENT_ID")
	if err != nil {
		return nil, err
	}

	retOptions.DiskSize, err = fromEnvOrError("DISK_SIZE")
	if err != nil {
		return nil, err
	}

	retOptions.DiskImage, err = fromEnvOrError("DISK_IMAGE")
	if err != nil {
		return nil, err
	}

	retOptions.MachineType, err = fromEnvOrError("MACHINE_TYPE")
	if err != nil {
		return nil, err
	}

	retOptions.Region, err = fromEnvOrError("REGION")
	if err != nil {
		return nil, err
	}

	retOptions.AvailabilityDomain, err = fromEnvOrError("AVAILABILITY_DOMAIN")
	if err != nil {
		return nil, err
	}

	return retOptions, nil
}

func fromEnvOrError(name string, fallback ...string) (string, error) {
	envvars := append([]string{name}, fallback...)

	for _, e := range envvars {
		val := os.Getenv(e)
		if val != "" {
			return val, nil
		}
	}

	envvarCsv := strings.Join(envvars, ", ")

	return "", fmt.Errorf("couldn't find option %s in environment, please make sure %s is defined", envvarCsv, envvarCsv)
}
