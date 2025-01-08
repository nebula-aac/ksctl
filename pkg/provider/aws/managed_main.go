// Copyright 2024 Ksctl Authors
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

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/ksctl/ksctl/pkg/consts"
)

const (
	assumeClusterRolePolicyDocument = `{
    "Version": "2012-10-17",
    "Statement": {
        "Sid": "TrustPolicyStatementThatAllowsEC2ServiceToAssumeTheAttachedRole",
        "Effect": "Allow",
        "Principal": { "Service": "eks.amazonaws.com" },
       "Action": "sts:AssumeRole"
    }
}`

	assumeWorkerNodeRolePolicyDocument = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "Service": "ec2.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}`
)

func (p *Provider) ManagedAddons(s addons.ClusterAddons) (externalCNI bool) {

	p.l.Debug(p.ctx, "Printing", "cni", s)
	addons := s.GetAddons("eks")

	switch consts.KsctlValidCNIPlugin(s) {
	case consts.CNICilium, consts.CNIFlannel:
		p.cni = s
		return false
	case "":
		p.cni = string(consts.CNIFlannel)
		return false
	default:
		p.cni = string(consts.CNINone)
		return true
	}
}

func (p *Provider) DelManagedCluster() error {
	if len(p.state.CloudInfra.Aws.ManagedClusterName) == 0 {
		p.l.Print(p.ctx, "Skipping deleting EKS cluster.")
		return nil
	}

	if len(p.state.CloudInfra.Aws.ManagedNodeGroupName) == 0 {
		p.l.Print(p.ctx, "Skipping deleting EKS node-group.")
	} else {
		p.l.Print(p.ctx, "Deleting the EKS node-group", "name", p.state.CloudInfra.Aws.ManagedNodeGroupName)
		nodeParameter := eks.DeleteNodegroupInput{
			ClusterName:   aws.String(p.state.CloudInfra.Aws.ManagedClusterName),
			NodegroupName: aws.String(p.state.CloudInfra.Aws.ManagedNodeGroupName),
		}

		_, err := p.client.BeginDeleteNodeGroup(p.ctx, &nodeParameter)
		if err != nil {
			return err
		}

		p.l.Success(p.ctx, "Deleted the EKS node-group", "name", p.state.CloudInfra.Aws.ManagedNodeGroupName)

		p.state.CloudInfra.Aws.ManagedNodeGroupName = ""
		p.state.CloudInfra.Aws.ManagedNodeGroupArn = ""
		p.state.CloudInfra.Aws.NoManagedNodes = 0
		p.state.CloudInfra.Aws.ManagedNodeSize = ""
		err = p.store.Write(p.state)
		if err != nil {
			return err
		}
	}

	p.l.Print(p.ctx, "Deleting the EKS cluster", "name", p.state.CloudInfra.Aws.ManagedClusterName)
	clusterPerimeter := eks.DeleteClusterInput{
		Name: aws.String(p.state.CloudInfra.Aws.ManagedClusterName),
	}

	_, err := p.client.BeginDeleteManagedCluster(p.ctx, &clusterPerimeter)
	if err != nil {
		return err
	}

	p.state.CloudInfra.Aws.ManagedClusterName = ""
	p.state.CloudInfra.Aws.ManagedClusterArn = ""
	err = p.store.Write(p.state)
	if err != nil {
		return err
	}

	iamParameter := iam.DeleteRoleInput{
		RoleName: aws.String(p.state.CloudInfra.Aws.IamRoleNameWP),
	}

	_, err = p.client.BeginDeleteIAM(p.ctx, &iamParameter, "worker")
	if err != nil {
		return err
	}

	p.state.CloudInfra.Aws.IamRoleNameWP = ""
	p.state.CloudInfra.Aws.IamRoleArnWP = ""
	err = p.store.Write(p.state)
	if err != nil {
		return err
	}

	iamParameter = iam.DeleteRoleInput{
		RoleName: aws.String(p.state.CloudInfra.Aws.IamRoleNameCN),
	}

	_, err = p.client.BeginDeleteIAM(p.ctx, &iamParameter, "controlplane")
	if err != nil {
		return err
	}

	p.state.CloudInfra.Aws.IamRoleNameCN = ""
	p.state.CloudInfra.Aws.IamRoleArnCN = ""
	err = p.store.Write(p.state)
	if err != nil {
		return err
	}

	p.l.Success(p.ctx, "Deleted the EKS cluster", "name", p.state.CloudInfra.Aws.ManagedClusterName)
	return p.store.Write(p.state)
}

func (p *Provider) NewManagedCluster(noOfNode int) error {
	name := <-p.chResName
	vmType := <-p.chVMType

	iamRoleControlPlane := fmt.Sprintf("ksctl-%s-cp-role", name)

	p.l.Print(p.ctx, "Creating a new EKS cluster", "name", p.state.CloudInfra.Aws.ManagedClusterName)

	if len(p.state.CloudInfra.Aws.ManagedClusterName) != 0 {
		p.l.Print(p.ctx, "skipped already created EKS cluster", "name", p.state.CloudInfra.Aws.ManagedClusterName)
	} else {

		if len(p.state.CloudInfra.Aws.IamRoleNameCN) == 0 {
			iamParameter := iam.CreateRoleInput{
				RoleName:                 aws.String(iamRoleControlPlane),
				AssumeRolePolicyDocument: aws.String(assumeClusterRolePolicyDocument),
			}
			iamRespCp, err := p.client.BeginCreateIAM(p.ctx, "controlplane", &iamParameter)
			if err != nil {
				return err
			}

			p.state.CloudInfra.Aws.IamRoleNameCN = *iamRespCp.Role.RoleName
			p.state.CloudInfra.Aws.IamRoleArnCN = *iamRespCp.Role.Arn

			err = p.store.Write(p.state)
			if err != nil {
				return err
			}

			p.l.Success(p.ctx, "created the EKS controlplane role", "name", p.state.CloudInfra.Aws.IamRoleNameCN)
		} else {
			p.l.Print(p.ctx, "skipped already created EKS controlplane role", "name", p.state.CloudInfra.Aws.IamRoleNameCN)
		}

		parameter := eks.CreateClusterInput{
			Name:    aws.String(name),
			RoleArn: aws.String(p.state.CloudInfra.Aws.IamRoleArnCN),
			ResourcesVpcConfig: &eksTypes.VpcConfigRequest{
				EndpointPrivateAccess: aws.Bool(true),
				EndpointPublicAccess:  aws.Bool(true),
				PublicAccessCidrs:     []string{"0.0.0.0/0"},
				SubnetIds:             p.state.CloudInfra.Aws.SubnetIDs,
			},
			KubernetesNetworkConfig: &eksTypes.KubernetesNetworkConfigRequest{
				IpFamily: eksTypes.IpFamilyIpv4,
			},
			AccessConfig: &eksTypes.CreateAccessConfigRequest{
				AuthenticationMode:                      eksTypes.AuthenticationModeApi,
				BootstrapClusterCreatorAdminPermissions: aws.Bool(true),
			},
			Version: aws.String(p.K8sVersion),
		}
		p.state.CloudInfra.Aws.B.KubernetesVer = p.K8sVersion
		p.state.BootstrapProvider = "managed"

		p.l.Print(p.ctx, "creating the EKS Controlplane")
		clusterResp, err := p.client.BeginCreateEKS(p.ctx, &parameter)
		if err != nil {
			return err
		}

		p.state.CloudInfra.Aws.ManagedClusterName = *clusterResp.Cluster.Name
		err = p.store.Write(p.state)
		if err != nil {
			return err
		}
	}
	iamRoleWorkerPlane := fmt.Sprintf("ksctl-%s-wp-role", p.state.CloudInfra.Aws.ManagedClusterName)

	if len(p.state.CloudInfra.Aws.ManagedNodeGroupName) != 0 {
		p.l.Print(p.ctx, "skipped already created EKS nodegroup", "name", p.state.CloudInfra.Aws.ManagedNodeGroupName)
	} else {
		if len(p.state.CloudInfra.Aws.IamRoleNameWP) == 0 {
			iamParameter := iam.CreateRoleInput{
				RoleName:                 aws.String(iamRoleWorkerPlane),
				AssumeRolePolicyDocument: aws.String(assumeWorkerNodeRolePolicyDocument),
			}
			iamRespWp, err := p.client.BeginCreateIAM(p.ctx, "worker", &iamParameter)
			if err != nil {
				return err
			}

			p.state.CloudInfra.Aws.IamRoleNameWP = *iamRespWp.Role.RoleName
			p.state.CloudInfra.Aws.IamRoleArnWP = *iamRespWp.Role.Arn
			err = p.store.Write(p.state)
			if err != nil {
				return err
			}

			p.l.Success(p.ctx, "created the EKS worker role", "name", p.state.CloudInfra.Aws.IamRoleNameWP)
		} else {
			p.l.Print(p.ctx, "skipped already created ROLE EKS Worker", "name", p.state.CloudInfra.Aws.IamRoleNameWP)
		}

		eksNodeGroupName := p.state.CloudInfra.Aws.ManagedClusterName + "-nodegroup"
		p.state.CloudInfra.Aws.ManagedNodeSize = vmType
		p.state.CloudInfra.Aws.NoManagedNodes = noOfNode

		nodegroup := eks.CreateNodegroupInput{
			ClusterName:   aws.String(p.state.CloudInfra.Aws.ManagedClusterName),
			NodeRole:      aws.String(p.state.CloudInfra.Aws.IamRoleArnWP),
			NodegroupName: aws.String(eksNodeGroupName),
			Subnets:       p.state.CloudInfra.Aws.SubnetIDs,
			CapacityType:  eksTypes.CapacityTypesOnDemand,

			InstanceTypes: []string{vmType},
			DiskSize:      aws.Int32(30),

			ScalingConfig: &eksTypes.NodegroupScalingConfig{
				DesiredSize: aws.Int32(int32(noOfNode)),
				MaxSize:     aws.Int32(int32(noOfNode)),
				MinSize:     aws.Int32(int32(noOfNode)),
			},
		}
		p.l.Print(p.ctx, "creating the EKS nodegroup")

		nodeResp, err := p.client.BeginCreateNodeGroup(p.ctx, &nodegroup)
		if err != nil {
			return err
		}
		p.state.CloudInfra.Aws.ManagedNodeGroupVmSize = vmType
		p.state.CloudInfra.Aws.ManagedNodeGroupName = *nodeResp.Nodegroup.NodegroupName
		p.state.CloudInfra.Aws.ManagedNodeGroupArn = *nodeResp.Nodegroup.NodegroupArn

		err = p.store.Write(p.state)
		if err != nil {
			return err
		}
		p.l.Success(p.ctx, "created the EKS nodegroup", "name", p.state.CloudInfra.Aws.ManagedNodeGroupName)

	}

	kubeconfig, err := p.client.GetKubeConfig(p.ctx, p.state.CloudInfra.Aws.ManagedClusterName)
	if err != nil {
		return err
	}

	p.state.CloudInfra.Aws.B.IsCompleted = true

	p.state.ClusterKubeConfig = kubeconfig
	p.state.ClusterKubeConfigContext = p.state.CloudInfra.Aws.ManagedClusterName

	if err := p.store.Write(p.state); err != nil {
		return err
	}

	return nil
}
