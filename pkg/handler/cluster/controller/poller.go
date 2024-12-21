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

package controller

import (
	"github.com/ksctl/ksctl/pkg/config"
	"github.com/ksctl/ksctl/pkg/logger"
	"github.com/ksctl/ksctl/pkg/validation"
	"runtime/debug"
	"sort"

	ksctlErrors "github.com/ksctl/ksctl/pkg/errors"

	"github.com/ksctl/ksctl/pkg/poller"

	localstate "github.com/ksctl/ksctl/pkg/storage/host"
	kubernetesstate "github.com/ksctl/ksctl/pkg/storage/kubernetes"
	externalmongostate "github.com/ksctl/ksctl/pkg/storage/mongodb"

	"github.com/ksctl/ksctl/pkg/consts"
)

func (manager *Controller) StartPoller() error {
	if _, ok := config.IsContextPresent(manager.ctx, consts.KsctlTestFlagKey); !ok {
		poller.InitSharedGithubReleasePoller()
	} else {
		poller.InitSharedGithubReleaseFakePoller(func(org, repo string) ([]string, error) {
			vers := []string{"v0.0.1"}

			if org == "etcd-io" && repo == "etcd" {
				vers = append(vers, "v3.5.15")
			}

			if org == "k3s-io" && repo == "k3s" {
				vers = append(vers, "v1.30.3+k3s1")
			}

			if org == "kubernetes" && repo == "kubernetes" {
				vers = append(vers, "v1.31.0")
			}

			sort.Slice(vers, func(i, j int) bool {
				return vers[i] > vers[j]
			})

			return vers, nil
		})
	}

	return nil
}

func (manager *Controller) InitStorage(p *Client) error {
	if !validation.ValidateStorage(p.Metadata.StateLocation) {
		return ksctlErrors.WrapError(
			ksctlErrors.ErrInvalidStorageProvider,
			manager.l.NewError(
				manager.ctx, "Problem in validation", "storage", p.Metadata.StateLocation,
			),
		)
	}
	switch p.Metadata.StateLocation {
	case consts.StoreLocal:
		p.Storage = localstate.NewClient(manager.ctx, manager.l)
	case consts.StoreExtMongo:
		p.Storage = externalmongostate.NewClient(manager.ctx, manager.l)
	case consts.StoreK8s:
		p.Storage = kubernetesstate.NewClient(manager.ctx, manager.l)
	}

	if err := p.Storage.Connect(); err != nil {
		return err
	}
	manager.l.Debug(manager.ctx, "initialized storageFactory")
	return nil
}

func (_ *Controller) PanicCatcher(log logger.Logger) {
	if r := recover(); r != nil {
		log.Error("Failed to recover stack trace", "error", r)
		debug.PrintStack()
	}
}
