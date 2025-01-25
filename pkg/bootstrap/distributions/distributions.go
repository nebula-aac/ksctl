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

package distributions

import (
	"github.com/ksctl/ksctl/pkg/addons"
	"github.com/ksctl/ksctl/pkg/consts"
)

type KubernetesDistribution interface {
	Setup(operation consts.KsctlOperation) error

	ConfigureControlPlane(int) error

	JoinWorkerplane(int) error

	K8sVersion(string) KubernetesDistribution

	CNI(addons.ClusterAddons) (externalCNI bool)
}
