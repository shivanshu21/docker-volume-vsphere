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
	"time"
	. "gopkg.in/check.v1"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/ssh"
	"github.com/vmware/docker-volume-vsphere/tests/constants/admincli"
)

const (
	volName1                 = "_test_vol_1"
	volName2                 = "_test_vol_2"
	defaultVg                = "_DEFAULT"
	defcname                 = "def_test_ctr"
	defVgName                = "def_test_vg"
)

type VmGroupTest struct {
	host string
	vmgroup string
	datastore1 string 
	datastore2 string
	cname string
	dockerHosts []string
}

var _ = Suite(&VmGroupTest{})

func curTime() string {
	return time.Now().Format(time.UnixDate)
}

func (vg *VmGroupTest) SetUpSuite(c *C) {
	vg.dockerHosts = []string{os.Getenv("VM1"), os.Getenv("VM2"), os.Getenv("VM3")}
	vg.host = os.Getenv("ESX")
	vg.vmgroup = os.Getenv("T1")
	vg.datastore1 = os.Getenv("DS1")
	vg.datastore2 = os.Getenv("DS2")
	vg.cname = os.Getenv("CNAME")

	if vg.vmgroup == "" {
		vg.vmgroup = defVgName
	}

	if vg.datastore1 == "" || vg.datastore2 == "" {
		os.Exit(1)
	}

	if vg.cname == "" {
		vg.cname = defcname
	}
}

func (vg *VmGroupTest) TearDownSuite(c *C) {

}

// Tests to validate behavior with the __DEFAULT_ .

func (vg *VmGroupTest) createVolumeOnDefaultVg(c *C, name string) {
	// 1. Create the volume on each host
	for _, vm := range vg.dockerHosts {
		_, err := dockercli.CreateVolume(vm, name)
		c.Assert(err, IsNil, Commentf("Error while creating volume - %s on VM - %s, err - %s", name, vm, err.Error()))
	}

	// 2. Verify the volume is created on the default vm group
	cmd := admincli.ListVolumes + " | grep " + defaultVg + " | grep " + name
	val, err := ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while listing volumes [%s] in default  on host %s [%s] err - %s", cmd, vg.host, err.Error()))
	c.Assert(val, Not(Equals), "", Commentf("Volume %s not found in default ", name))
}

// TestVolumeCreateOnDefaultVg - Verify that volumes can be created on the
// default volume group with default permissions, then attached and deleted
// Assumes: VM (VM1) belongs to the default VM group.
// 1. Create a volume in the default vmgroup
// 2. Verify the VM is able to attach and run a container with the volume
// 3. Delete the volume
func (vg *VmGroupTest) TestVolumeCreateOnDefaultVg(c *C) {
	c.Logf("START: vmgroups-test.TestVolumeCreateOnDefaultVg %s", curTime())
	// Create a volume in the default group
	vg.createVolumeOnDefaultVg(c, volName1)

	// 1. Verify volume can be mounted and used on at least one of the VMs
	_, err := dockercli.AttachVolume(vg.dockerHosts[0], volName1, vg.cname)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s, err - %s", volName1, vg.dockerHosts[0], err.Error()))

	// 2. Delete the volume in the default group
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], volName1)
	c.Assert(err, IsNil, Commentf("Error while deleting volume - %s in the default , err - %s", volName1, err.Error()))
	c.Logf("Passed - Volume create and attach on default ")
	c.Logf("STOP: vmgroups-test.TestVolumeCreateOnDefaultVg %s", curTime())
}

// TestVolumeAccessAcrossVmGroups - Verify volumes can be accessed only
// from VMs that belong to the volume group
// Assumes: VMs (VM1i and VM2) belongs to the default VM group.
// 1. Create a volume in the default VM group from VM1
// 2. Create a new vmgroup and add VM1 to it with vg.datastore2 as its default
// 3. Try attaching the volume created in the default group from VM1 - expect error
// 4. Try deleteing the volume in the default group from VM1 - expect error
// 5. Try deleting the volume in th default group from VM2
// 6. Remove the newly created vmgroup 
func (vg *VmGroupTest) TestVolumeAccessAcrossVmGroups(c *C) {
	c.Logf("START: vmgroups-test.TestVolumeAccessAcrossVmGroups %s", curTime())
	// 1. Create a volume in the default group
	vg.createVolumeOnDefaultVg(c, volName1)

	// 2. Create a new vmgroup, with vg.datastore2 as the datastore
	cmd := admincli.CreateVmGroup + vg.vmgroup + " --vm-list " + vg.dockerHosts[0] + " --default-datastore " + vg.datastore2
	_, err := ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while creating volume group - [%s] err - %s", cmd, err.Error()))

	// 3. Try to attach the volume created in the default volume group
	_, err = dockercli.AttachVolume(vg.dockerHosts[0], volName1, vg.cname)
	c.Assert(err, Not(IsNil), Commentf("Expected error when attaching volume %s in default  from VM %s in vmgroup %s", volName1, vg.dockerHosts[0], vg.vmgroup))

	// 4. Try deleting volume in default group
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], volName1)
	c.Assert(err, Not(IsNil), Commentf("Expected error when deleting volume %s in default  from VM %s in vmgroup %s", volName1, vg.dockerHosts[0], vg.vmgroup))

	// 5. Remove the volume from the default , from the other VM
	_, err = dockercli.DeleteVolume(vg.dockerHosts[1], volName1)
	c.Assert(err, IsNil, Commentf("Error when deleting volume %s in default  from VM %s in vmgroup %s", volName1, vg.dockerHosts[1], vg.vmgroup))

	// 6. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMsForVmGroup + vg.vmgroup  + " --vm-list " + vg.dockerHosts[0]
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing VM %s from  on host %s - [%s], err %s", vg.dockerHosts[1], vg.host, cmd, err.Error()))

	// 7. Remove the new vmgroup
	cmd = admincli.DeleteVmGroup + vg.vmgroup
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing vmgroup [%s] on host %s - [%s], err %s", vg.dockerHosts[1], vg.host, cmd, err.Error()))

	c.Logf("Passed - Volume access across s")
	c.Logf("STOP: vmgroups-test.TestVolumeAccessAcrossVmGroups %s", curTime())
}

// TestCreateAccessPrivilegeOnDefaultVg - Verify volumes can be
// created by a VM as long as the vmgroup has the allow-create setting
// enabled on it
// Assumes: VM1 is in the default vmgroup
// 1. Create volume in default group from vm VM1
// 2. Try attaching volume from VM1 and run a container
// 3. Remove the create privilege from the default vmgroup
// 4. Try create a volume in the default vmgroup - expect error
// 5. Restore create privilege on default vmgroup
// 6. Remove volume created in (1).
func (vg *VmGroupTest) TestCreateAccessPrivilegeOnDefaultVg(c *C) {
	c.Logf("START: vmgroups-test.TestCreateAccessPrivilegeOnDefaultVg %s", curTime())
	// 1. Create a volume in the default group
	vg.createVolumeOnDefaultVg(c, volName1)

	// 2. Attach volume from default 
	_, err := dockercli.AttachVolume(vg.dockerHosts[0], volName1, vg.cname)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s, err - %s", volName1, vg.dockerHosts[0], err.Error()))

	// 3. Remove the create privilege on the default  for specified datastore
	cmd := admincli.ModifyAccessForVmGroup + defaultVg + "--allow-create False --datastore " + vg.datastore1
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting access privileges [%s] on default  on host %s, err - %s", cmd, vg.host, err.Error()))

	// 4. Try creating a volume on the default 
	_, err = dockercli.CreateVolume(vg.dockerHosts[0], volName2)
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, on default , err - %s", volName2, vg.dockerHosts[0], err.Error()))

	// 5. Restore the create privilege on the default  for specified datastore
	cmd = admincli.ModifyAccessForVmGroup + defaultVg + "--allow-create True --datastore " + vg.datastore1
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when restoring access privileges [%s] on default  on host %s, err - %s", cmd, vg.host, err.Error()))

	// 6. Remove the volume created earlier
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], volName1)
	c.Assert(err, IsNil, Commentf("Error while deleting volume - %s in the default , err - %s", volName1, err.Error()))
	c.Logf("Passed - create privilege on default ")
	c.Logf("STOP: vmgroups-test.TestCreateAccessPrivilegeOnDefaultVg %s", curTime())
}

// TestVolumeCreateOnVg - Verify basic volume create/attach/delete
// on non-default vmgroup
// 1. Create a new vmgroup and place VM VM1 in it
// 2. Create volume in vmgroup
// 3. Attach volume and run a container
// 4. Delete volume created in (2)
// 5. Destroy the VM group
func (vg *VmGroupTest) TestVolumeCreateOnVg(c *C) {
	c.Logf("START: vmgroups-test.TestVolumeCreateOnVg %s", curTime())
	// 1. Create a new  vmgroup
	cmd := admincli.CreateVmGroup + vg.vmgroup + " --vm-list " + vg.dockerHosts[0] + " --default-datastore " + vg.datastore1
	_, err := ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while creating volume group - [%s] err - %s", cmd, err.Error()))

	// 2. Create a volume in the new vmgroup
	_, err = dockercli.CreateVolume(vg.dockerHosts[0], volName2)
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	// 3. Try attaching volume in new vmgroup
	_, err = dockercli.AttachVolume(vg.dockerHosts[0], volName2, vg.cname)
	c.Assert(err, IsNil, Commentf("Error while attaching volume - %s on VM - %s, err - %s", volName1, vg.dockerHosts[0], err.Error()))

	// 4. Remove the volume from the new vmgroup
	_, err = dockercli.DeleteVolume(vg.dockerHosts[0], volName2)
	c.Assert(err, IsNil, Commentf("Expected error when deleting volume %s in default  from VM %s in vmgroup %s", volName2, vg.dockerHosts[0], vg.vmgroup))

	// 5. Remove the VM from the new vmgroup
	cmd = admincli.RemoveVMsForVmGroup + vg.vmgroup + " --vm-list " + vg.dockerHosts[0]
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when removing VM %s from  on host %s - [%s], err %s", vg.dockerHosts[1], vg.host, cmd, err.Error()))

	// 6. Remove the new  vmgroup
	cmd = admincli.DeleteVmGroup + vg.vmgroup
	ssh.ExecCmd(vg.host, cmd)

	c.Logf("Passed - create and attach volumes on a non-default ")
	c.Logf("STOP: vmgroups-test.TestVolumeCreateOnVg %s", curTime())
}

// TestVerifyMaxFileSizeOnVg - Verify that enough volumes can be created
// to match the totalsize for a vmgroup and verify that volumes of the
// maxsize can be created.
// 1. Create a VM group and make vg.datastore1 as its default
// 2. Set maxsize and totalsize to 1G each in the new vmgroup
// 3. Try creating a volume of 1gb
// 4. Try creating another volume of 1gb, 1023mb, 1024mb, 1025mb - expect error
// 5. Set maxsize and total size as 1gb and 2gb respectively
// 6. Retry step (4) - expect success this time
// 7. Remove both volumes
// 8. Remove the vmgroup created in (1)
func (vg *VmGroupTest) TestVerifyMaxFileSizeOnVg(c *C) {
	c.Logf("START: vmgroups-test.TestVerifyMaxFileSizeOnVg %s", curTime())
	// 1. Create a  and add VM to it
	cmd := admincli.CreateVmGroup + vg.vmgroup + " --default-datastore " + vg.datastore1
	_, err := ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error while creating volume group - [%s] err - %s", cmd, err.Error()))

	// 2. Ensure the max file size and total size is set to 1G each.
	cmd = admincli.ModifyAccessForVmGroup + defaultVg + " --datastore " + vg.datastore1 + " --volume-maxsize=1gb --volume-totalsize=1gb --allow-create=True"
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting max and total size [%s] on  %s on host %s, err - %s", cmd, vg.vmgroup, vg.host, err.Error()))

	// 3. Try creating volumes up to the max filesize and the totalsize
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName1, "-o size=1gb")
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s, err - %s", volName1, vg.dockerHosts[0], err.Error()))

	// 4. Try creating a volume of 1gb again, should fail as totalsize is already reached
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName2, "-o size=1gb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName2, "-o size=1023mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName2, "-o size=1024mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName2, "-o size=1025mb")
	c.Assert(err, Not(IsNil), Commentf("Expected error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	// 7. Ensure the max file size and total size is set to 1G each.
	cmd = admincli.ModifyAccessForVmGroup + defaultVg + " --datastore " + vg.datastore1 + " --volume-maxsize=1gb --volume-totalsize=2gb --allow-create=True"
	_, err = ssh.ExecCmd(vg.host, cmd)
	c.Assert(err, IsNil, Commentf("Error when setting max and total size [%s] on  %s on host %s, err - %s", cmd, vg.vmgroup, vg.host, err.Error()))

	// 8. Try creating a volume of 1gb again, should succeed as totalsize is increased to 2gb
	_, err = dockercli.CreateVolumeWithOpts(vg.dockerHosts[0], volName2, "-o size=1024mb")
	c.Assert(err, IsNil, Commentf("Error while creating volume - %s from VM - %s, err - %s", volName2, vg.dockerHosts[0], err.Error()))

	// 9. Delete both volumes
	dockercli.DeleteVolume(vg.dockerHosts[0], volName1)
	dockercli.DeleteVolume(vg.dockerHosts[0], volName2)

	// 10. Remove the vmgroup
	cmd = admincli.DeleteVmGroup + vg.vmgroup
	ssh.ExecCmd(vg.host, cmd)

	c.Logf("Passed - verified volumes can be created to match total size assigned to a ")
	c.Logf("STOP: vmgroups-test.TestVerifyMaxFileSizeOnVg %s", curTime())
}

func (vg *VmGroupTest) TestVolumeVisibilityOnVg(c *C) {
	c.Skip("Not supported")
}
