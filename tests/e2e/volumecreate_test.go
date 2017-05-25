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

// This test is going to cover various volume creation test cases

package e2e

import (
	"log"
	"os"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"

	. "gopkg.in/check.v1"
)

const ErrorVolumeCreate = "Error response from daemon: create"

type VolumeCreateTestSuite struct {
	hostIP     string
	esxIP      string
	dsNameList []string
	volumeList []string
}

func (s *VolumeCreateTestSuite) SetUpSuite(c *C) {
	s.hostIP = os.Getenv("VM2")
	s.esxIP = os.Getenv("ESX")
	s.dsNameList = govc.GetDatastoreList()
	log.Printf("Datastores found are %v", s.dsNameList)
}

func (s *VolumeCreateTestSuite) SetUpTest(c *C) {
	s.volumeList = s.volumeList[:0]
}

func (s *VolumeCreateTestSuite) TearDownTest(c *C) {
	for _, name := range s.volumeList {
		out, err := dockercli.DeleteVolume(s.hostIP, name)
		c.Assert(err, IsNil, Commentf(out))
	}
}

var _ = Suite(&VolumeCreateTestSuite{})

// Valid volume names test
// 1. having 100 chars
// 2. having various chars including alphanumerics
// 3. ending in 5Ns
// 4. ending in 7Ns
// 5. contains @datastore (valid name)
// 6. contains multiple '@'
// 7. contains unicode character
func (s *VolumeCreateTestSuite) TestValidName(c *C) {

	s.volumeList = append(s.volumeList, inputparams.GetVolumeNameOfSize(100))
	s.volumeList = append(s.volumeList, "Volume-0000000-****-###")
	s.volumeList = append(s.volumeList, "Volume-00000")
	s.volumeList = append(s.volumeList, "Volume-0000000")
	s.volumeList = append(s.volumeList, inputparams.GetVolumeNameWithTimeStamp("abc")+"@"+s.dsNameList[0])
	s.volumeList = append(s.volumeList, inputparams.GetVolumeNameWithTimeStamp("abc")+"@@@@"+s.dsNameList[0])
	s.volumeList = append(s.volumeList, inputparams.GetVolumeNameWithTimeStamp("Volume-你"))

	for _, name := range s.volumeList {
		out, err := dockercli.CreateVolume(s.hostIP, name)
		c.Assert(err, IsNil, Commentf(out))

		isAvailable := verification.CheckVolumeAvailability(s.hostIP, name)
		c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", name))
	}
}

// Invalid volume names test
// 1. having more than 100 chars
// 2. ending -NNNNNN (6Ns)
// 3. contains @invalid datastore name
func (s *VolumeCreateTestSuite) TestInvalidName(c *C) {
	var invalidVolList []string

	invalidVolList = append(invalidVolList, inputparams.GetVolumeNameOfSize(101))
	invalidVolList = append(invalidVolList, "Volume-000000")
	invalidVolList = append(invalidVolList, inputparams.GetVolumeNameWithTimeStamp("Volume")+"@invalidDatastore")

	for _, name := range invalidVolList {
		out, _ := dockercli.CreateVolume(s.hostIP, name)
		c.Assert(strings.HasPrefix(out, ErrorVolumeCreate), Equals, true)
	}
}

// Valid volume creation options
// 1. size 10gb
// 2. disk format (thin, zeroedthick, eagerzeroedthick)
// 3. attach-as (persistent, independent_persistent)
// 4. fstype ext4
// 5. access (read-write, read-only)
// 6. clone-from valid volume
// 7. fstype xfs
func (s *VolumeCreateTestSuite) TestValidOptions(c *C) {
	var validVolOpts []string

	validVolOpts = append(validVolOpts, " -o size=10gb")
	validVolOpts = append(validVolOpts, " -o diskformat=zeroedthick")
	validVolOpts = append(validVolOpts, " -o diskformat=thin")
	validVolOpts = append(validVolOpts, " -o diskformat=eagerzeroedthick")
	validVolOpts = append(validVolOpts, " -o attach-as=independent_persistent")
	validVolOpts = append(validVolOpts, " -o attach-as=persistent")
	validVolOpts = append(validVolOpts, " -o fstype=ext4")
	validVolOpts = append(validVolOpts, " -o access=read-only")
	validVolOpts = append(validVolOpts, " -o access=read-write")

	// Need a valid volume source to test clone-from option
	cloneSrcVol := inputparams.GetVolumeNameWithTimeStamp("clone_src")
	s.volumeList = append(s.volumeList, cloneSrcVol)
	out, err := dockercli.CreateVolume(s.hostIP, cloneSrcVol)
	c.Assert(err, IsNil, Commentf(out))
	validVolOpts = append(validVolOpts, " -o clone-from="+cloneSrcVol)

	for _, option := range validVolOpts {
		volName := inputparams.GetVolumeNameWithTimeStamp("valid_opts")
		s.volumeList = append(s.volumeList, volName)

		// Create volume with options
		out, err := dockercli.CreateVolumeWithOptions(s.hostIP, volName, option)
		c.Assert(err, IsNil, Commentf(out))

		// Check the availability of volume
		isAvailable := verification.CheckVolumeAvailability(s.hostIP, volName)
		c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", volName))
	}

	// xfs file system needs volume name upto than 12 characters
	xfsVolName := inputparams.GetVolumeNameOfSize(12)
	s.volumeList = append(s.volumeList, xfsVolName)
	out, err = dockercli.CreateVolumeWithOptions(s.hostIP, xfsVolName, " -o fstype=xfs")
	c.Assert(err, IsNil, Commentf(out))
	isAvailable := verification.CheckVolumeAvailability(s.hostIP, xfsVolName)
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", xfsVolName))
}

// Invalid volume create operations
// 1. Wrong disk formats
// 2. Wrong volume sizes
// 3. Wrong fs types
// 4. Wrong access types
// 5. Unavailable clone source
func (s *VolumeCreateTestSuite) TestInvalidOptions(c *C) {
	var invalidVolOpts []string

	invalidVolOpts = append(invalidVolOpts, " -o diskformat=zeroedthickk")
	invalidVolOpts = append(invalidVolOpts, " -o diskformat=zeroedthick,thin")
	invalidVolOpts = append(invalidVolOpts, " -o size=100mbb")
	invalidVolOpts = append(invalidVolOpts, " -o size=100gbEE")
	invalidVolOpts = append(invalidVolOpts, " -o sizes=100mb")
	invalidVolOpts = append(invalidVolOpts, " -o fstype=xfs_ext")
	invalidVolOpts = append(invalidVolOpts, " -o access=read-write-both")
	invalidVolOpts = append(invalidVolOpts, " -o access=write-only")
	invalidVolOpts = append(invalidVolOpts, " -o access=read-write-both")
	invalidVolOpts = append(invalidVolOpts, " -o clone-from=IDontExist")

	for _, option := range invalidVolOpts {
		volName := inputparams.GetVolumeNameWithTimeStamp("invalid_opts")

		out, _ := dockercli.CreateVolumeWithOptions(s.hostIP, volName, option)
		c.Assert(strings.HasPrefix(out, ErrorVolumeCreate), Equals, true)
	}
}
