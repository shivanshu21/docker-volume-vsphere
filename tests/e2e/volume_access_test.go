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

// The goal of this test suite is to verify read/write consistency on volumes
// in accordance with the access updates on the volume
package e2e

import (
	"log"
	"os"
	"strings"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
)

const ErrorWriteVolume = "Read-only file system"

type VolumeAccessTestSuite struct {
	volumeName    string
	dockerHostIP  []string
	containerList [2][]string
	esxIP         string
}

func (s *VolumeAccessTestSuite) SetUpSuite(c *C) {
	s.dockerHostIP = []string{os.Getenv("VM1"), os.Getenv("VM2")}
	s.esxIP = os.Getenv("ESX")

	dsName := govc.GetDatastoreList()[0]
	s.volumeName = inputparams.GetVolumeNameWithTimeStamp("vol_access") + "@" + dsName

	// Create a volume
	out, err := dockercli.CreateVolume(s.dockerHostIP[0], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

func (s *VolumeAccessTestSuite) TearDownTest(c *C) {
	for i := 0; i < 2; i++ {
		for _, cname := range s.containerList[i] {
			out, err := dockercli.RemoveContainer(s.dockerHostIP[i], cname)
			c.Assert(err, IsNil, Commentf(out))
		}
	}

	out, err := dockercli.DeleteVolume(s.dockerHostIP[0], s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

var _ = Suite(&VolumeAccessTestSuite{})

func (s *VolumeAccessTestSuite) newCName(i int) string {
	cname := inputparams.GetContainerNameWithTimeStamp("vol_access")
	s.containerList[i] = append(s.containerList[i], cname)
	return cname
}

// Verify read, write is possible after volume access update
// 1. Write a message from host1 to a file on the volume
// 2. Read the content from host2 from same file on the volume
//    Verify the content is same.
// 3. Write another message from host2 to the same file on that volume
// 4. Update the volume access to read-only
// 5. Write from host1 to the file on volume should fail
// 6. Write from host2 should also fail
// 7. Update the volume access to read-write
// 8. Write from host1 should succeed
// 9. Write from host2 should succeed
func (s *VolumeAccessTestSuite) TestAccessUpdate(c *C) {
	log.Printf("START: volume_access_test.TestAccessUpdate")

	data1 := "message_by_host1"
	data2 := "message_by_host2"
	testFile := "test.txt"

	out, err := dockercli.WriteToVolume(s.dockerHostIP[0], s.volumeName, s.newCName(0), testFile, data1)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.ReadFromVolume(s.dockerHostIP[1], s.volumeName, s.newCName(1), testFile)
	c.Assert(err, IsNil, Commentf(out))

	c.Assert(out, Equals, data1)

	out, err = dockercli.WriteToVolume(s.dockerHostIP[1], s.volumeName, s.newCName(1), testFile, data2)
	c.Assert(err, IsNil, Commentf(out))

	out, err = admincli.UpdateVolumeAccess(s.esxIP, s.volumeName, "_DEFAULT", "read-only")
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.WriteToVolume(s.dockerHostIP[0], s.volumeName, s.newCName(0), testFile, data1)
	c.Assert(strings.Contains(out, ErrorWriteVolume), Equals, true, Commentf(out))

	out, err = dockercli.WriteToVolume(s.dockerHostIP[1], s.volumeName, s.newCName(1), testFile, data2)
	c.Assert(strings.Contains(out, ErrorWriteVolume), Equals, true, Commentf(out))

	out, err = admincli.UpdateVolumeAccess(s.esxIP, s.volumeName, "_DEFAULT", "read-write")
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.WriteToVolume(s.dockerHostIP[0], s.volumeName, s.newCName(0), testFile, data1)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.WriteToVolume(s.dockerHostIP[1], s.volumeName, s.newCName(1), testFile, data2)
	c.Assert(err, IsNil, Commentf(out))

	log.Printf("END: volume_access_test.TestAccessUpdate")
}
