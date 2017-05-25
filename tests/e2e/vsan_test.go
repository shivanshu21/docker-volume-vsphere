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

// This test is going to cover various vsan related test cases

package e2e

import (
	"os"
	"strings"

	"github.com/vmware/docker-volume-vsphere/tests/utils/admincli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/govc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"

	. "gopkg.in/check.v1"
)

type VsanTestSuite struct {
	hostIP     string
	esxIP      string
	vsanDSName string
	volumeList []string
}

func (s *VsanTestSuite) SetUpSuite(c *C) {
	s.hostIP = os.Getenv("VM2")
	s.esxIP = os.Getenv("ESX")

	dsNameList := govc.GetDatastoreList()
	s.vsanDSName = ""
	for _, ds := range dsNameList {
		if strings.HasPrefix(ds, "vsan") {
			s.vsanDSName = ds
		}
	}
}

func (s *VsanTestSuite) SetUpTest(c *C) {
	s.volumeList = s.volumeList[:0]
}

func (s *VsanTestSuite) TearDownTest(c *C) {
	for _, name := range s.volumeList {
		out, err := dockercli.DeleteVolume(s.hostIP, name)
		c.Assert(err, IsNil, Commentf(out))
	}
}

var _ = Suite(&VsanTestSuite{})

func (s *VsanTestSuite) TestVSANPolicy(c *C) {
	if s.vsanDSName == "" {
		c.Skip("Vsan datastore unavailable")
	}

	policyName := "validPolicy"
	out, err := admincli.CreatePolicy(s.esxIP, policyName, "'((\"proportionalCapacity\" i50)''(\"hostFailuresToTolerate\" i0))'")
	c.Assert(err, IsNil, Commentf(out))

	invalidContentPolicyName := "invalidPolicy"
	out, err = admincli.CreatePolicy(s.esxIP, invalidContentPolicyName, "'((\"wrongKey\" i50)'")
	c.Assert(err, IsNil, Commentf(out))

	volName := inputparams.GetVolumeNameWithTimeStamp("vsanVol") + "@" + s.vsanDSName
	s.volumeList = append(s.volumeList, volName)
	vsanOpts := " -o vsan-policy-name=" + policyName

	out, err = dockercli.CreateVolumeWithOptions(s.hostIP, volName, vsanOpts)
	c.Assert(err, IsNil, Commentf(out))
	isAvailable := verification.CheckVolumeAvailability(s.hostIP, volName)
	c.Assert(isAvailable, Equals, true, Commentf("Volume %s is not available after creation", volName))

	invalidVsanOpts := [2]string{"-o vsan-policy-name=IDontExist", "-o vsan-policy-name=" + invalidContentPolicyName}
	for _, option := range invalidVsanOpts {
		volName = inputparams.GetVolumeNameWithTimeStamp("vsanVol") + "@" + s.vsanDSName
		out, _ = dockercli.CreateVolumeWithOptions(s.hostIP, volName, option)
		c.Assert(strings.HasPrefix(out, ErrorVolumeCreate), Equals, true)
	}
}
