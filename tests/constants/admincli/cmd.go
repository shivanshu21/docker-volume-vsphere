// Copyright 2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// A home to hold test constants related with vmdkops_admin cli.

package admincli

const (
	// location of the vmdkops binary
	vmdkopsAdmin = "/usr/lib/vmware/vmdkops/bin/vmdkops_admin.py "

	// vmdkops_admin volume
	vmdkopsAdminVolume = vmdkopsAdmin + "volume "

	// ListVolumes referring to vmdkops_admin volume ls
	ListVolumes = vmdkopsAdminVolume + "ls "

	// CreatePolicy Create a policy
	CreatePolicy = vmdkopsAdmin + " policy create "

	// SetVolumeAccess set volume access
	SetVolumeAccess = vmdkopsAdminVolume + " set "
)
