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

// This test is going to create volume on the fresh testbed very first time.
// After installing vmdk volume plugin/driver, volume creation should not be
// failed very first time.

// This test is going to cover the issue reported at #656
// TODO: as of now we are running the test against photon vm it should be run
// against various/applicable linux distros.

package e2e

import (
	"os"
	"strings"
	. "gopkg.in/check.v1"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
)

const (
	defaultVg                = "_DEFAULT"
	defContainer             = "VmGroup_Container"
	defVgName                = "vmgroup_test"
	vmgroupsTest             = "vmgroup"
)

type VmGroupTest struct {
	host string
	vmgroup string
	datastore1 string
	datastore2 string
	vgContainer string
	dockerHosts []string
	dockerHostNames []string
	volName1 string
	volName2 string
	volName3 string
}

var _ = Suite(&VmGroupTest{})

func (vg *VmGroupTest) SetUpSuite(c *C) {
	vg.dockerHosts = append(vg.dockerHosts, os.Getenv("VM1"))
	vg.dockerHosts = append(vg.dockerHosts, os.Getenv("VM2"))
	vg.dockerHostNames = append(vg.dockerHostNames, os.Getenv("VM1NAME"))
	vg.dockerHostNames = append(vg.dockerHostNames, os.Getenv("VM2NAME"))
	vg.host = inputparams.GetEsxIP()
	vg.vmgroup = defVgName
	vg.datastore1 = "_VM_DS"
	vg.datastore2 = os.Getenv("DS1")
	vg.vgContainer = inputparams.GetContainerNameWithTimeStamp(defContainer)

	if vg.vmgroup == "" {
		vg.vmgroup = defVgName
	}

	if vg.datastore1 == "" || vg.datastore2 == "" {
		c.Skip("Unknown or missing datastores for test, skipping vmgroup tests.")
	}

	if vg.vgContainer == "" {
		vg.vgContainer = defContainer
	}
}

func (vg *VmGroupTest) TearDownSuite(c *C) {
}

func (vg *VmGroupTest) SetUpTest(c *C) {
	cmd := admincli.CreateVMgroup + vg.vmgroup + " --default-datastore " + vg.datastore2
	out, err := ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf(out))
}

func (vg *VmGroupTest) TearDownTest(c *C) {
	cmd := admincli.AccessSetVMgroup + defaultVg + " --allow-create True --datastore " + vg.datastore1
	ssh.InvokeCommand(vg.host, cmd)

	cmd = admincli.RemoveVMFromVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	ssh.InvokeCommand(vg.host, cmd)

	cmd = admincli.RemoveVMgroup + vg.vmgroup + " --remove-volumes"
	ssh.InvokeCommand(vg.host, cmd)
}

func (vg *VmGroupTest) vmgroupGetVolName(c *C) {
	vg.volName1 = inputparams.GetVolumeNameWithTimeStamp(vmgroupsTest)
	vg.volName2 = inputparams.GetVolumeNameWithTimeStamp(vmgroupsTest)
	vg.volName3 = inputparams.GetVolumeNameWithTimeStamp(vmgroupsTest)
}

// Tests to validate behavior with the __DEFAULT_ vmgroup.

func (vg *VmGroupTest) createVolumeOnDefaultVg(c *C, name string) {
	// 1. Create the volume on host
	_, err := dockercli.CreateVolume(vg.dockerHosts[0], name)
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s on VM - %s", name, vg.dockerHosts[0]))

	// 2. Verify the volume is created on the default vm group
	val, err := dockercli.ListVolumes(vg.dockerHosts[0])
	c.Assert(err, IsNil, Commentf("Error while listing volumes [%s] in default  on host [%s]", vg.dockerHosts[0]))
	c.Assert(strings.Contains(val, name), Equals, true, Commentf("Volume %s not found in default vmgroup", name))
}

// TestVmGroupVolumeCreateOnDefaultVg - Verify that volumes can be created on the
// default vmgroup with default permissions, then attached and deleted
// Assumes: VM (VM1) belongs to the default VM group.
// 1. Create a volume in the default vmgroup
// 2. Verify the VM is able to attach and run a container with the volume
// 3. Delete the volume
func (vg *VmGroupTest) TestVmGroupVolumeCreateOnDefaultVg(c *C) {
	misc.LogTestStart(c, vmgroupsTest, "TestVmGroupVolumeCreateOnDefaultVg")

	vg.vmgroupGetVolName(c)
	// Create a volume in the default group
	vg.createVolumeOnDefaultVg(c, vg.volName1)

	// 1. Verify volume can be mounted
	_, err := dockercli.AttachVolume(vg.dockerHosts[0], vg.volName1, vg.vgContainer)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s", vg.volName1, vg.dockerHosts[0]))

	// Docker may not have completed the detach yet with the host.
	misc.SleepForSec(2)

	// 2. Delete the volume in the default group
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName1)
	c.Assert(err, IsNil, Commentf("Error while deleting volume - %s in the default vmgroup", vg.volName1))
	c.Logf("Passed - Volume create and attach on default vmgroup")

	misc.LogTestEnd(c, vmgroupsTest, "TestVmGroupVolumeCreateOnDefaultVg")
}

// TestVmGroupVolumeAccessAcrossVmGroups - Verify volumes can be accessed only
// from VMs that belong to the vmgroup
// Assumes: VMs (VM1i and VM2) belongs to the default VM group.
// 1. Create a volume in the default VM group from VM1
// 2. Create a new vmgroup and add VM1 to it with vg.datastore2 as its default
// 3. Try attaching the volume created in the default group from VM1 - expect error
// 4. Try deleteing the volume in the default group from VM1 - expect error
// 5. Try deleting the volume in th default group from VM2
// 6. Remove the newly created vmgroup
func (vg *VmGroupTest) TestVmGroupVolumeAccessAcrossVmGroups(c *C) {
	misc.LogTestStart(c, vmgroupsTest, "TestVmGroupVolumeAccessAcrossVmGroups")
	vg.vmgroupGetVolName(c)

	// 1. Create a volume in the default group
	vg.createVolumeOnDefaultVg(c, vg.volName1)

	// 2. Create a new vmgroup, with vg.datastore2 as the datastore
	cmd := admincli.AddVMToVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	_, err := ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while adding VM to vmgroup - [%s]", cmd))

	// 3. Try to inspect the volume created in the default vmgroup, trying to run a container
	// causes Docker to figure the volume isn't there and creates a local volume.
	_, err = dockercli.InspectVolume(vg.dockerHosts[0], vg.volName1)
	c.Assert(err, Not(IsNil), Commentf("Expected error inspecting volume %s default vmgroup from VM %s in vmgroup %s", vg.volName1, vg.dockerHosts[0], vg.vmgroup))

	// 4. Try deleting volume in default group
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName1)
	c.Assert(err, Not(IsNil), Commentf("Expected error when deleting volume %s in default vmgroup from VM %s in vmgroup %s", vg.volName1, vg.dockerHosts[0], vg.vmgroup))

	// 5. Remove the volume from the default , from the other VM
	_, err = dockercli.DeleteVolume(vg.dockerHosts[1], vg.volName1)
	c.Assert(err, IsNil, Commentf("Error when deleting volume %s in default  from VM %s in vmgroup %s", vg.volName1, vg.dockerHosts[1], vg.vmgroup))

	// 6. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMFromVMgroup + vg.vmgroup  + " --vm-list " + vg.dockerHostNames[0]
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing VM %s from  on host %s - [%s]", vg.dockerHosts[1], vg.host, cmd))

	// 7. Remove the new vmgroup
	cmd = admincli.RemoveVMgroup + vg.vmgroup
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing vmgroup [%s] on host %s - [%s]", vg.dockerHosts[1], vg.host, cmd))

	c.Logf("Passed - Volume access across vmgroups")
	misc.LogTestEnd(c, vmgroupsTest, "TestVmGroupVolumeAccessAcrossVmGroups")
}

// TestVmGroupCreateAccessPrivilegeOnDefaultVg - Verify volumes can be
// created by a VM as long as the vmgroup has the allow-create setting
// enabled on it
// Assumes: VM1 is in the default vmgroup
// 1. Create volume in default group from vm VM1
// 2. Try attaching volume from VM1 and run a container
// 3. Remove the create privilege from the default vmgroup
// 4. Try create a volume in the default vmgroup - expect error
// 5. Restore create privilege on default vmgroup
// 6. Remove volume created in (1).
func (vg *VmGroupTest) TestVmGroupCreateAccessPrivilegeOnDefaultVg(c *C) {
	misc.LogTestStart(c, vmgroupsTest, "TestVmGroupCreateAccessPrivilegeOnDefaultVg")

	vg.vmgroupGetVolName(c)

	// 1. Create a volume in the default vmgroup
	vg.createVolumeOnDefaultVg(c, vg.volName1)

	// 2. Attach volume from default vmgroup
	_, err := dockercli.AttachVolume(vg.dockerHosts[0], vg.volName1, vg.vgContainer)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s", vg.volName1, vg.dockerHosts[0]))

	// 3. Remove the create privilege on the default  for specified datastore
	cmd := admincli.AccessSetVMgroup + defaultVg + " --allow-create False --datastore " + vg.datastore1
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting access privileges [%s] on default  on host %s", cmd, vg.host))

	cmd = admincli.AccessGetVMgroup + defaultVg + " | grep " + vg.dockerHosts[0] + " | grep False"
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, Not(IsNil), Commentf("Expected create access to be turned off [%s] on default vmgroup on host %s", cmd, vg.host))

	// 4. Try creating a volume on the default vmgroup
	_, err = dockercli.CreateVolume(vg.dockerHosts[0], vg.volName2)
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, on default", vg.volName2, vg.dockerHosts[0]))

	// 5. Restore the create privilege on the default  for specified datastore
	cmd = admincli.AccessSetVMgroup + defaultVg + " --allow-create True --datastore " + vg.datastore1
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when restoring access privileges [%s] on default vmgroup on host %s", cmd, vg.host))

	// 6. Remove the volume created earlier
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName1)
	c.Assert(err, IsNil, Commentf("Error while deleting volume - %s in the default vmgroup", vg.volName1))
	c.Logf("Passed - create privilege on default vmgroup")
	misc.LogTestEnd(c, vmgroupsTest, "TestVmGroupCreateAccessPrivilegeOnDefaultVg")
}

// TestVmGroupVolumeCreateOnVg - Verify basic volume create/attach/delete
// on non-default vmgroup
// 1. Create a new vmgroup and place VM VM1 in it
// 2. Create volume in vmgroup
// 3. Attach volume and run a container
// 4. Delete volume created in (2)
// 5. Destroy the VM group
func (vg *VmGroupTest) TestVmGroupVolumeCreateOnVg(c *C) {
	misc.LogTestStart(c, vmgroupsTest, "TestVmGroupVolumeCreateOnVg")
	vg.vmgroupGetVolName(c)

	// 1. Add VM to new  vmgroup
	cmd := admincli.AddVMToVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	_, err := ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while adding VM to vmgroup - [%s]", cmd))

	// 2. Create a volume in the new vmgroup
	_, err = dockercli.CreateVolume(vg.dockerHosts[0], vg.volName2)
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))

	// 3. Try attaching volume in new vmgroup
	_, err = dockercli.AttachVolume(vg.dockerHosts[0], vg.volName2, vg.vgContainer)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s", vg.volName2, vg.dockerHosts[0]))

	// Docker may not have completed the detach yet with the host.
	misc.SleepForSec(2)

	// 4. Remove the volume from the new vmgroup
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName2)
	c.Assert(err, IsNil, Commentf("Expected error when deleting volume %s in default  from VM %s in vmgroup %s", vg.volName2, vg.dockerHosts[0], vg.vmgroup))

	// 5. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMFromVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing VM %s from  on host %s - [%s]", vg.dockerHosts[1], vg.host, cmd))

	c.Logf("Passed - create and attach volumes on a non-default vmgroup")
	misc.LogTestEnd(c, vmgroupsTest, "TestVmGroupVolumeCreateOnVg")
}

// TestVmGroupVerifyMaxFileSizeOnVg - Verify that enough volumes can be created
// to match the totalsize for a vmgroup and verify that volumes of the
// maxsize can be created.
// 1. Create a VM group and make vg.datastore2 as its default
// 2. Set maxsize and totalsize to 1G each in the new vmgroup
// 3. Try creating a volume of 1gb
// 4. Try creating another volume of 1gb, 1023mb, 1024mb, 1025mb - expect error
// 5. Set maxsize and total size as 1gb and 2gb respectively
// 6. Retry step (4) - expect success this time
// 7. Remove both volumes
// 8. Remove the vmgroup created in (1)
func (vg *VmGroupTest) TestVmGroupVerifyMaxFileSizeOnVg(c *C) {
	misc.LogTestStart(c, vmgroupsTest, "TestVmGroupVerifyMaxFileSizeOnVg")
	vg.vmgroupGetVolName(c)

	// 1. Add VM to vmgroup
	cmd := admincli.AddVMToVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	_, err := ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while adding to  vmgroup - [%s]", cmd))

	// 2. Ensure the max file size and total size is set to 1G each.
	cmd = admincli.AccessSetVMgroup + vg.vmgroup + " --datastore " + vg.datastore2 + " --volume-maxsize=1gb --volume-totalsize=1gb --allow-create=True"
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting max and total size [%s] on  %s on host %s", cmd, vg.vmgroup, vg.host))

	// 3. Try creating volumes up to the max filesize and the totalsize
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName1, "-o size=1gb")
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s", vg.volName1, vg.dockerHosts[0]))

	// 4. Try creating a volume of 1gb again, should fail as totalsize is already reached
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName2, "-o size=1gb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))

	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName3, "-o size=1023mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))

/*
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName2, "-o size=1024mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))

	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName2, "-o size=1025mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))
*/

	// 5. Ensure the max file size and total size is set to 1G and 2G each.
	cmd = admincli.AccessSetVMgroup + vg.vmgroup + " --datastore " + vg.datastore2 + " --volume-maxsize=1gb --volume-totalsize=2gb --allow-create=True"
	_, err = ssh.InvokeCommand(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting max and total size [%s] on  %s on host %s", cmd, vg.vmgroup, vg.host))

	// 6. Try creating a volume of 1gb again, should succeed as totalsize is increased to 2gb
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], vg.volName2, "-o size=1024mb")
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s", vg.volName2, vg.dockerHosts[0]))

	// 7. Delete both volumes
	dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName1)
	dockercli.DeleteVolume(vg.dockerHosts[0], vg.volName2)

	// 8. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMFromVMgroup + vg.vmgroup + " --vm-list " + vg.dockerHostNames[0]
	_, err = ssh.InvokeCommand(vg.host, cmd)

	c.Logf("Passed - verified volumes can be created to match total size assigned to a vmgroup")
	misc.LogTestEnd(c, vmgroupsTest, "TestVmGroupVerifyMaxFileSizeOnVg")
}

func (vg *VmGroupTest) TestVmGroupVolumeVisibilityOnVg(c *C) {
	c.Skip("Not supported")
}
