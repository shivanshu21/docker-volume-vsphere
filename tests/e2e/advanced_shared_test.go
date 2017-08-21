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
        "strconv"
	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/misc"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
	. "gopkg.in/check.v1"
)

const (
        // Name of the test file in shared volume
        testFileName = "test.txt"
)

type AdvancedSharedTestSuite struct {
	config          *inputparams.TestConfig
	esx             string
	master          string
	worker1         string
	volName1        string
	container1Name  string
	container2Name  string
}

func (s *AdvancedSharedTestSuite) SetUpSuite(c *C) {
	s.config = inputparams.GetTestConfig()
	if s.config == nil {
		c.Skip("Unable to retrieve test config, skipping advanced shared tests")
	}

	s.esx = s.config.EsxHost
	s.master = inputparams.GetSwarmManager1()
        s.worker1 = inputparams.GetSwarmWorker1()
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

        data := []string{"QWERTYUIOP000000000000",
                         "ASDFGHJKLL111111111111"}
        // Create shared volume
	out, err := dockercli.CreateSharedVolume(s.worker1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

        // Check if the shared volume got created properly
	accessible := verification.CheckVolumeAvailability(s.worker1, s.volName1)
	c.Assert(accessible, Equals, true, Commentf("Volume %s is not available", s.volName1))

        // Mount the volume on master
        out, err = dockercli.AttachSharedVolume(s.master, s.volName1, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        // Expect global refcount for this volume to be 1
        out = verification.GetSharedVolumeGlobalRefcount(s.volName1, s.master)
        grefc, _ := strconv.Atoi(out)
        c.Assert(grefc, Equals, 1, Commentf("Expected volume global refcount to be 1, found %s", out))

        // Mount the volume on worker
        out, err = dockercli.AttachSharedVolume(s.worker1, s.volName1, s.container1Name)

        // Expect global refcount for this volume to be 2
        out = verification.GetSharedVolumeGlobalRefcount(s.volName1, s.master)
        grefc, _ = strconv.Atoi(out)
        c.Assert(grefc, Equals, 2, Commentf("Expected volume global refcount to be 2, found %s", out))


        // Try IO from both VMs and verify the written data
        s.readWriteCheck(c, s.master, s.worker1, data[0])
        s.readWriteCheck(c, s.worker1, s.master, data[1])

        // Unmount shared volume from master
        out, err = dockercli.RemoveContainer(s.master, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        // Expect global refcount for this volume to be 1
        out = verification.GetSharedVolumeGlobalRefcount(s.volName1, s.master)
        grefc, _ = strconv.Atoi(out)
        c.Assert(grefc, Equals, 1, Commentf("Expected volume global refcount to be 1, found %s", out))

        // Unmount shared volume from worker
        out, err = dockercli.RemoveContainer(s.worker1, s.container1Name)
        c.Assert(err, IsNil, Commentf(out))

        // Expect global refcount for this volume to be 0
        out = verification.GetSharedVolumeGlobalRefcount(s.volName1, s.master)
        grefc, _ = strconv.Atoi(out)
        c.Assert(grefc, Equals, 0, Commentf("Expected volume global refcount to be 0, found %s", out))

        // delete the volume // Will uncomment after unmount() code is done
	out, err = dockercli.DeleteVolume(s.worker1, s.volName1)
	c.Assert(err, IsNil, Commentf(out))

	misc.LogTestEnd(c.TestName())
}

// readWriteCheck Writes data to shared volume from one VM and read from another.
// Fails if the data is not identical.
func (s *AdvancedSharedTestSuite) readWriteCheck(c *C, node1 string, node2 string, data string) {
        out, err := dockercli.WriteToVolume(node1, s.volName1, s.container2Name, testFileName, data)
        c.Assert(err, IsNil, Commentf(out))

        out, err = dockercli.ReadFromVolume(node2, s.volName1, s.container2Name, testFileName)
        c.Assert(err, IsNil, Commentf(out))

        mismatchCondition := (out != data)
        c.Assert(mismatchCondition, Equals, false,
            Commentf("Volume data inconsistent! Wrote: %s, Read: %s", data, out))
        return
}
