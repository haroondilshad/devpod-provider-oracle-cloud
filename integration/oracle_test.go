//go:build integration

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

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/oracle"
	"github.com/haroondilshad/devpod-provider-oracle-cloud/pkg/options"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateAndDeleteInstance tests the full lifecycle of an instance
// This test requires valid OCI credentials and will create and delete real resources
func TestCreateAndDeleteInstance(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Get options from environment
	opts, err := options.FromEnv(false)
	require.NoError(t, err)

	// Create a unique machine ID for this test
	opts.MachineID = "test-" + time.Now().Format("20060102-150405")

	// Create configuration provider
	configProvider, err := oracle.CreateOCIConfigurationProvider(opts.OCIConfigFile, opts.OCIProfile)
	require.NoError(t, err)

	// Create Oracle client
	o, err := oracle.NewOracle(configProvider)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate SSH key pair
	publicKey, privateKey, err := client.CreateSSHKeyPair()
	require.NoError(t, err)

	// Create instance options
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
	require.NoError(t, err)

	// Launch instance
	_, err = o.computeClient.LaunchInstance(ctx, *request)
	require.NoError(t, err)

	// Cleanup at the end of the test
	defer func() {
		err := o.DeleteInstance(ctx, opts.MachineID)
		assert.NoError(t, err)

		// Wait for instance to be deleted
		for i := 0; i < 30; i++ {
			status, err := o.GetInstanceStatus(ctx, opts.MachineID)
			if err != nil || status == "not_found" {
				break
			}
			time.Sleep(10 * time.Second)
		}
	}()

	// Wait for instance to be running
	var status string
	for i := 0; i < 30; i++ {
		status, err = o.GetInstanceStatus(ctx, opts.MachineID)
		require.NoError(t, err)
		if status == "running" {
			break
		}
		time.Sleep(10 * time.Second)
	}
	assert.Equal(t, "running", status)

	// Get instance IP
	ip, err := o.GetInstanceIP(ctx, opts.MachineID)
	require.NoError(t, err)
	assert.NotEmpty(t, ip)

	// Test stopping the instance
	err = o.StopInstance(ctx, opts.MachineID)
	require.NoError(t, err)

	// Wait for instance to be stopped
	for i := 0; i < 30; i++ {
		status, err = o.GetInstanceStatus(ctx, opts.MachineID)
		require.NoError(t, err)
		if status == "stopped" {
			break
		}
		time.Sleep(10 * time.Second)
	}
	assert.Equal(t, "stopped", status)

	// Test starting the instance
	err = o.StartInstance(ctx, opts.MachineID)
	require.NoError(t, err)

	// Wait for instance to be running again
	for i := 0; i < 30; i++ {
		status, err = o.GetInstanceStatus(ctx, opts.MachineID)
		require.NoError(t, err)
		if status == "running" {
			break
		}
		time.Sleep(10 * time.Second)
	}
	assert.Equal(t, "running", status)
} 