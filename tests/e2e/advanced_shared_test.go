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

// This test suite includes test cases to verify advanced shared volume
// cases including multiple writers writing data to the same volume

// +build runonceshared

package e2e

import (
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

const (
        // Data that will be written to the test file in shared volume
        data = "1234567890ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        // Name of the test file in shared volume
        testFileName = "test.txt"
)

type AdvancedSharedTestSuite struct {
	config          *inputparams.TestConfig
	esx             string
	mgr1            string
	mgr2            string
	mgr3            string
	volName1        string
	container1Name  string
	container2Name  string
}

func (s *AdvancedSharedTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping basic sharedtests")
	}

	s.esx = s.config.EsxHost
	s.mgr1 = inputparams.GetSwarmManager1()
        s.mgr2 = inputparams.GetSwarmWorker1()
        s.mgr3 = inputparams.GetSwarmWorker2()
}

func (s *AdvancedSharedTestSuite) SetUpTest(c *C) {
	s.volName1 = inputparams.GetSharedVolumeName()
	s.container1Name = inputparams.GetUniqueContainerName(c.TestName())
	s.container2Name = inputparams.GetUniqueContainerName(c.TestName())
}

var _ = Suite(&AdvancedSharedTestSuite{})

// TestSharedVolumeLifecycle -  Creates shared volume, mounts it on
// two different host VMs. Runs IO on each and verifies the written
// result.
func (s *AdvancedSharedTestSuite) TestSharedVolumeLifecycle(c *C) {
	misc.LogTestStart(c.TestName())

	out, err := dockercli.CreateSharedVolume(s.mgr2, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	accessible := verification.CheckVolumeAvailability(s.mgr2, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

        out, err = dockercli.AttachSharedVolume(s.mgr1, s.volName1, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        out, err = dockercli.AttachSharedVolume(s.mgr2, s.volName1, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        // Try IO from both VMs and verify the written data
        s.readWriteCheck(c, s.mgr1, s.mgr2)
        s.readWriteCheck(c, s.mgr2, s.mgr1)

        out, err = dockercli.RemoveContainer(s.mgr1, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        out, err = dockercli.RemoveContainer(s.mgr2, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        // delete the volume //<<<< Will uncomment after unmount() code is done
	//out, err = dockercli.DeleteVolume(s.mgr3, s.volName1)
	//c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// readWriteCheck Writes data to shared volume from one VM and read from another.
// Fails if the data is not identical.
func (s *AdvancedSharedTestSuite) readWriteCheck(c *C, node1 string, node2 string) {
        out, err := dockercli.WriteToVolume(node1, s.volName1, s.container2Name, testFileName, data)
        c.Assert(err, IsNil, Commentf(out))

        out, err = dockercli.ReadFromVolume(node2, s.volName1, s.container2Name, testFileName)
        c.Assert(err, IsNil, Commentf(out))

        mismatchCondition := (out != data)
        c.Assert(mismatchCondition, Equals, false,
            Commentf("Volume data inconsistent! Wrote: %s, Read: %s", data, out))
        return
}
