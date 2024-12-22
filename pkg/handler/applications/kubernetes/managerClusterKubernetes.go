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

package kubernetes

//package provisioner
//
//import (
//	"context"
//
//	bootstrapController "github.com/ksctl/ksctl/pkg/controllers/bootstrap"
//	cloudController "github.com/ksctl/ksctl/pkg/controllers/cloud"
//	"github.com/ksctl/ksctl/pkg/helpers/consts"
//	"github.com/ksctl/ksctl/pkg/types"
//	storageTypes "github.com/ksctl/ksctl/pkg/types/storage"
//)
//
//type ManagerClusterKubernetes struct {
//	managerInfo
//}
//
//func NewManagerClusterKubernetes(ctx context.Context, log types.LoggerFactory, client *types.KsctlClient) (*ManagerClusterKubernetes, error) {
//	defer panicCatcher(log)
//
//	stateDocument = new(storageTypes.StorageDocument)
//	controllerCtx = context.WithValue(ctx, consts.KsctlModuleNameKey, "ksctl-manager")
//
//	cloudController.InitLogger(controllerCtx, log)
//	bootstrapController.InitLogger(controllerCtx, log)
//
//	manager := new(ManagerClusterKubernetes)
//	manager.log = log
//	manager.client = client
//
//	if err := manager.initStorage(controllerCtx); err != nil {
//		log.Error("handled error", "catch", err)
//		return nil, err
//	}
//
//	if err := manager.startPoller(controllerCtx); err != nil {
//		log.Error("handled error", "catch", err)
//		return nil, err
//	}
//
//	return manager, nil
//}
//
//func (manager *ManagerClusterKubernetes) ApplicationsAndCni(op consts.KsctlOperation) error {
//
//	client := manager.client
//	log := manager.log
//	defer panicCatcher(log)
//
//	if err := manager.setupConfigurations(); err != nil {
//		log.Error("handled error", "catch", err)
//		return err
//	}
//
//	if client.Metadata.Provider == consts.CloudLocal {
//		client.Metadata.Region = "LOCAL"
//	}
//
//	clusterType := consts.ClusterTypeMang
//	if client.Metadata.IsHA {
//		clusterType = consts.ClusterTypeHa
//	}
//
//	if err := client.Storage.Setup(
//		client.Metadata.Provider,
//		client.Metadata.Region,
//		client.Metadata.ClusterName,
//		clusterType); err != nil {
//		log.Error("handled error", "catch", err)
//		return err
//	}
//
//	defer func() {
//		if err := client.Storage.Kill(); err != nil {
//			log.Error("StorageClass Kill failed", "reason", err)
//		}
//	}()
//
//	if err := cloudController.InitCloud(client, stateDocument, consts.OperationGet); err != nil {
//		log.Error("handled error", "catch", err)
//		return err
//	}
//
//	if op != consts.OperationCreate && op != consts.OperationDelete {
//
//		err := log.NewError(controllerCtx, "Invalid operation")
//		log.Error("handled error", "catch", err)
//		return err
//	}
//
//	if err := bootstrapController.ApplicationsInCluster(client, stateDocument, op); err != nil {
//		log.Error("handled error", "catch", err)
//		return err
//	}
//	return nil
//}
