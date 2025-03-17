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

package oracle

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"text/template"
	"time"

	cryptoSsh "golang.org/x/crypto/ssh"

	"github.com/google/uuid"
	"github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/ssh"
	"github.com/loft-sh/log"
	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/core"
	"github.com/oracle/oci-go-sdk/v65/identity"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

//go:embed cloud-config.yaml
var cloudConfig embed.FS

type cloudInit struct {
	Status string `json:"status"`
}

type Oracle struct {
	computeClient    *core.ComputeClient
	networkClient    *core.VirtualNetworkClient
	identityClient   *identity.IdentityClient
}

func NewOracle(configProvider common.ConfigurationProvider) (*Oracle, error) {
	computeClient, err := core.NewComputeClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}

	networkClient, err := core.NewVirtualNetworkClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}

	identityClient, err := identity.NewIdentityClientWithConfigurationProvider(configProvider)
	if err != nil {
		return nil, err
	}

	return &Oracle{
		computeClient:    &computeClient,
		networkClient:    &networkClient,
		identityClient:   &identityClient,
	}, nil
}

func (o *Oracle) upsertPublicKey(ctx context.Context, publicKey, machineID string) (*core.InstanceSourceViaImageDetails, error) {
	fingerprint, err := generateSSHKeyFingerprint(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate fingerprint for public ssh key")
	}

	// Generate name
	if len(machineID) >= 24 {
		machineID = machineID[:24]
	}
	name := fmt.Sprintf("%s-%s", machineID, uuid.NewString()[:8])

	log.Default.Infof("Creating instance with SSH key: %s", name)

	// Create cloud-init data
	cloudInitData, err := o.generateCloudConfig(publicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate cloud config")
	}

	// Encode cloud-init data
	encodedCloudInitData := base64.StdEncoding.EncodeToString([]byte(cloudInitData))

	// Create instance source details
	sourceDetails := &core.InstanceSourceViaImageDetails{
		ImageId: nil, // Will be set later
		LaunchOptions: &core.LaunchOptions{
			BootVolumeType:                 core.InstanceSourceBootVolumeTypeParavirtualized,
			NetworkType:                    core.InstanceLaunchOptionsNetworkTypeParavirtualized,
			IsConsistentVolumeNamingEnabled: common.Bool(true),
		},
		KmsKeyId: nil,
	}

	return sourceDetails, nil
}

func (o *Oracle) BuildInstanceOptions(
	ctx context.Context,
	machineID string,
	diskImage string,
	diskSizeGB string,
	machineType string,
	region string,
	availabilityDomain string,
	compartmentID string,
	publicKey string,
) (*core.LaunchInstanceRequest, error) {
	// Get source details
	sourceDetails, err := o.upsertPublicKey(ctx, publicKey, machineID)
	if err != nil {
		return nil, err
	}

	// Find image
	image, err := o.findImage(ctx, compartmentID, diskImage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find image")
	}
	sourceDetails.ImageId = image.Id

	// Create or get VCN and subnet
	vcn, subnet, err := o.createOrGetNetwork(ctx, compartmentID, availabilityDomain, machineID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create or get network")
	}

	// Parse disk size
	diskSize, err := strconv.Atoi(diskSizeGB)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse disk size")
	}

	// Create instance request
	request := &core.LaunchInstanceRequest{
		LaunchInstanceDetails: core.LaunchInstanceDetails{
			AvailabilityDomain: &availabilityDomain,
			CompartmentId:      &compartmentID,
			Shape:              &machineType,
			DisplayName:        common.String(fmt.Sprintf("devpod-%s", machineID)),
			SourceDetails:      sourceDetails,
			CreateVnicDetails: &core.CreateVnicDetails{
				SubnetId:       subnet.Id,
				AssignPublicIp: common.Bool(true),
			},
			Metadata: map[string]string{
				"user_data": base64.StdEncoding.EncodeToString([]byte(o.generateCloudConfig(publicKey))),
			},
			FreeformTags: map[string]string{
				labelMachineID: machineID,
				labelType:      labelTypeDevPod,
			},
		},
	}

	return request, nil
}

func (o *Oracle) findImage(ctx context.Context, compartmentID, diskImage string) (*core.Image, error) {
	request := core.ListImagesRequest{
		CompartmentId: &compartmentID,
		DisplayName:   common.String(diskImage),
	}

	response, err := o.computeClient.ListImages(ctx, request)
	if err != nil {
		return nil, err
	}

	if len(response.Items) == 0 {
		return nil, fmt.Errorf("image %s not found", diskImage)
	}

	return &response.Items[0], nil
}

func (o *Oracle) createOrGetNetwork(ctx context.Context, compartmentID, availabilityDomain, machineID string) (*core.Vcn, *core.Subnet, error) {
	// Check if VCN exists
	listVcnRequest := core.ListVcnsRequest{
		CompartmentId: &compartmentID,
	}
	vcnResponse, err := o.networkClient.ListVcns(ctx, listVcnRequest)
	if err != nil {
		return nil, nil, err
	}

	var vcn *core.Vcn
	for _, v := range vcnResponse.Items {
		if *v.DisplayName == "devpod-vcn" {
			vcn = &v
			break
		}
	}

	// Create VCN if it doesn't exist
	if vcn == nil {
		createVcnRequest := core.CreateVcnRequest{
			CreateVcnDetails: core.CreateVcnDetails{
				CompartmentId: &compartmentID,
				DisplayName:   common.String("devpod-vcn"),
				CidrBlock:     common.String("10.0.0.0/16"),
				DnsLabel:      common.String("devpodvcn"),
				FreeformTags: map[string]string{
					labelType: labelTypeDevPod,
				},
			},
		}
		vcnResponse, err := o.networkClient.CreateVcn(ctx, createVcnRequest)
		if err != nil {
			return nil, nil, err
		}
		vcn = &vcnResponse.Vcn
	}

	// Check if internet gateway exists
	listIgRequest := core.ListInternetGatewaysRequest{
		CompartmentId: &compartmentID,
		VcnId:         vcn.Id,
	}
	igResponse, err := o.networkClient.ListInternetGateways(ctx, listIgRequest)
	if err != nil {
		return nil, nil, err
	}

	var ig *core.InternetGateway
	for _, i := range igResponse.Items {
		if *i.DisplayName == "devpod-ig" {
			ig = &i
			break
		}
	}

	// Create internet gateway if it doesn't exist
	if ig == nil {
		createIgRequest := core.CreateInternetGatewayRequest{
			CreateInternetGatewayDetails: core.CreateInternetGatewayDetails{
				CompartmentId: &compartmentID,
				DisplayName:   common.String("devpod-ig"),
				VcnId:         vcn.Id,
				IsEnabled:     common.Bool(true),
				FreeformTags: map[string]string{
					labelType: labelTypeDevPod,
				},
			},
		}
		igResponse, err := o.networkClient.CreateInternetGateway(ctx, createIgRequest)
		if err != nil {
			return nil, nil, err
		}
		ig = &igResponse.InternetGateway
	}

	// Check if route table exists
	listRtRequest := core.ListRouteTablesRequest{
		CompartmentId: &compartmentID,
		VcnId:         vcn.Id,
	}
	rtResponse, err := o.networkClient.ListRouteTables(ctx, listRtRequest)
	if err != nil {
		return nil, nil, err
	}

	var rt *core.RouteTable
	for _, r := range rtResponse.Items {
		if *r.DisplayName == "devpod-rt" {
			rt = &r
			break
		}
	}

	// Create route table if it doesn't exist
	if rt == nil {
		createRtRequest := core.CreateRouteTableRequest{
			CreateRouteTableDetails: core.CreateRouteTableDetails{
				CompartmentId: &compartmentID,
				DisplayName:   common.String("devpod-rt"),
				VcnId:         vcn.Id,
				RouteRules: []core.RouteRule{
					{
						NetworkEntityId: ig.Id,
						Destination:     common.String("0.0.0.0/0"),
						DestinationType: core.RouteRuleDestinationTypeCidrBlock,
					},
				},
				FreeformTags: map[string]string{
					labelType: labelTypeDevPod,
				},
			},
		}
		rtResponse, err := o.networkClient.CreateRouteTable(ctx, createRtRequest)
		if err != nil {
			return nil, nil, err
		}
		rt = &rtResponse.RouteTable
	}

	// Check if subnet exists
	listSubnetRequest := core.ListSubnetsRequest{
		CompartmentId: &compartmentID,
		VcnId:         vcn.Id,
	}
	subnetResponse, err := o.networkClient.ListSubnets(ctx, listSubnetRequest)
	if err != nil {
		return nil, nil, err
	}

	var subnet *core.Subnet
	for _, s := range subnetResponse.Items {
		if *s.DisplayName == "devpod-subnet" {
			subnet = &s
			break
		}
	}

	// Create subnet if it doesn't exist
	if subnet == nil {
		createSubnetRequest := core.CreateSubnetRequest{
			CreateSubnetDetails: core.CreateSubnetDetails{
				CompartmentId:      &compartmentID,
				DisplayName:        common.String("devpod-subnet"),
				VcnId:              vcn.Id,
				CidrBlock:          common.String("10.0.0.0/24"),
				RouteTableId:       rt.Id,
				DnsLabel:           common.String("devpodsubnet"),
				AvailabilityDomain: &availabilityDomain,
				FreeformTags: map[string]string{
					labelType: labelTypeDevPod,
				},
			},
		}
		subnetResponse, err := o.networkClient.CreateSubnet(ctx, createSubnetRequest)
		if err != nil {
			return nil, nil, err
		}
		subnet = &subnetResponse.Subnet
	}

	return vcn, subnet, nil
}

func (o *Oracle) generateCloudConfig(publicKey string) (string, error) {
	// Read cloud-config template
	cloudConfigBytes, err := cloudConfig.ReadFile("cloud-config.yaml")
	if err != nil {
		return "", err
	}

	// Create template
	tmpl, err := template.New("cloud-config").Parse(string(cloudConfigBytes))
	if err != nil {
		return "", err
	}

	// Create template data
	data := struct {
		PublicKey string
		AgentB64  string
	}{
		PublicKey: publicKey,
		AgentB64:  "", // Will be filled by DevPod
	}

	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (o *Oracle) GetInstance(ctx context.Context, machineID string) (*core.Instance, error) {
	if machineID == "" {
		return nil, MissingMachineID()
	}

	// List instances with the machine ID tag
	request := core.ListInstancesRequest{
		DisplayName: common.String(fmt.Sprintf("devpod-%s", machineID)),
	}

	response, err := o.computeClient.ListInstances(ctx, request)
	if err != nil {
		return nil, err
	}

	// Find the instance with the matching machine ID
	for _, instance := range response.Items {
		if instance.FreeformTags[labelMachineID] == machineID {
			return &instance, nil
		}
	}

	return nil, MissingServer()
}

func (o *Oracle) DeleteInstance(ctx context.Context, machineID string) error {
	instance, err := o.GetInstance(ctx, machineID)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}
		return err
	}

	// Terminate instance
	request := core.TerminateInstanceRequest{
		InstanceId:         instance.Id,
		PreserveBootVolume: common.Bool(false),
	}

	_, err = o.computeClient.TerminateInstance(ctx, request)
	if err != nil && !IsNotFound(err) {
		return err
	}

	return nil
}

func (o *Oracle) StartInstance(ctx context.Context, machineID string) error {
	instance, err := o.GetInstance(ctx, machineID)
	if err != nil {
		return err
	}

	// Check if instance is already running
	if instance.LifecycleState == core.InstanceLifecycleStateRunning {
		return nil
	}

	// Start instance
	request := core.InstanceActionRequest{
		InstanceId: instance.Id,
		Action:     core.InstanceActionActionStart,
	}

	_, err = o.computeClient.InstanceAction(ctx, request)
	return err
}

func (o *Oracle) StopInstance(ctx context.Context, machineID string) error {
	instance, err := o.GetInstance(ctx, machineID)
	if err != nil {
		return err
	}

	// Check if instance is already stopped
	if instance.LifecycleState == core.InstanceLifecycleStateStopped {
		return nil
	}

	// Stop instance
	request := core.InstanceActionRequest{
		InstanceId: instance.Id,
		Action:     core.InstanceActionActionStop,
	}

	_, err = o.computeClient.InstanceAction(ctx, request)
	return err
}

func (o *Oracle) GetInstanceStatus(ctx context.Context, machineID string) (string, error) {
	instance, err := o.GetInstance(ctx, machineID)
	if err != nil {
		if IsNotFound(err) {
			return "not_found", nil
		}
		return "", err
	}

	switch instance.LifecycleState {
	case core.InstanceLifecycleStateRunning:
		return "running", nil
	case core.InstanceLifecycleStateStopped:
		return "stopped", nil
	case core.InstanceLifecycleStateTerminated:
		return "terminated", nil
	case core.InstanceLifecycleStateProvisioning, core.InstanceLifecycleStateStarting:
		return "starting", nil
	case core.InstanceLifecycleStateStopping:
		return "stopping", nil
	default:
		return string(instance.LifecycleState), nil
	}
}

func (o *Oracle) GetInstanceIP(ctx context.Context, machineID string) (string, error) {
	instance, err := o.GetInstance(ctx, machineID)
	if err != nil {
		return "", err
	}

	// Get VNIC attachments
	vnicRequest := core.ListVnicAttachmentsRequest{
		InstanceId: instance.Id,
	}

	vnicResponse, err := o.computeClient.ListVnicAttachments(ctx, vnicRequest)
	if err != nil {
		return "", err
	}

	if len(vnicResponse.Items) == 0 {
		return "", fmt.Errorf("no VNIC attachments found for instance %s", *instance.Id)
	}

	// Get VNIC
	vnicID := vnicResponse.Items[0].VnicId
	vnicGetRequest := core.GetVnicRequest{
		VnicId: vnicID,
	}

	vnicGetResponse, err := o.networkClient.GetVnic(ctx, vnicGetRequest)
	if err != nil {
		return "", err
	}

	// Return public IP if available, otherwise private IP
	if vnicGetResponse.Vnic.PublicIp != nil {
		return *vnicGetResponse.Vnic.PublicIp, nil
	}

	return *vnicGetResponse.Vnic.PrivateIp, nil
}

func generateSSHKeyFingerprint(publicKey string) (string, error) {
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		return "", err
	}

	return cryptoSsh.FingerprintSHA256(pubKey), nil
}

func CreateOCIConfigurationProvider(configFilePath, profile string) (common.ConfigurationProvider, error) {
	if configFilePath == "" {
		return nil, fmt.Errorf("OCI config file path is required")
	}

	if profile == "" {
		profile = "DEFAULT"
	}

	configProvider := common.CustomProfileConfigProvider(configFilePath, profile)
	return configProvider, nil
} 