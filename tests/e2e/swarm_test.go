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

// This test suite includes test cases to verify basic vDVS functionality
// in docker swarm mode.

package e2e

import (
	"log"

	. "gopkg.in/check.v1"

	"github.com/vmware/docker-volume-vsphere/tests/utils/dockercli"
	"github.com/vmware/docker-volume-vsphere/tests/utils/inputparams"
	"github.com/vmware/docker-volume-vsphere/tests/utils/verification"
)

type SwarmTestSuite struct {
	esxName     string
	master      string
	worker1     string
	worker2     string
	dockerHosts []string
	volumeName  string
	serviceName string
}

func (s *SwarmTestSuite) SetUpTest(c *C) {
	s.esxName = inputparams.GetEsxIP()
	s.master = inputparams.GetSwarmManager1()
	s.worker1 = inputparams.GetSwarmWorker1()
	s.worker2 = inputparams.GetSwarmWorker2()
	s.dockerHosts = []string{s.master, s.worker1, s.worker2}

	s.volumeName = inputparams.GetVolumeNameWithTimeStamp("swarm_test")
	s.serviceName = inputparams.GetServiceNameWithTimeStamp("swarm_test")

	// Verify if swarm cluster is already initialized
	out, err := dockercli.ListNodes(s.master)
	c.Assert(err, IsNil, Commentf(out))

	// Create the volume
	out, err = dockercli.CreateVolume(s.master, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyDetachedStatus(s.volumeName, s.master, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is not detached", s.volumeName))
}

func (s *SwarmTestSuite) TearDownTest(c *C) {
	// Clean up the volume
	out, err := dockercli.DeleteVolume(s.master, s.volumeName)
	c.Assert(err, IsNil, Commentf(out))
}

var _ = Suite(&SwarmTestSuite{})

// Test vDVS usage in swarm mode
//
// Test steps:
// 1. Create a docker service with replicas setting to 1
// 2. Verify the service is up and running with one node
// 3. Verify one container is spawned
// 5. Verify the volume is in attached status
// 6. Scale the service to set replica numbers to 2
// 7. Verify the service is up and running with two nodes
// 8. Verify 2 containers are spawned
// 9. Stop one node of the service
// 10. Verify the service is still running with two nodes
// 11. Verify there are still 2 containers up and running
// 12. Verify the volume is in attached status
// 13. Delete the volume - expect fail
// 14. Remove the service
// 15. Verify the service is gone
// 16. Verify the volume is in detached status
func (s *SwarmTestSuite) TestDockerSwarm(c *C) {
	log.Printf("START: swarm_test.TestDockerSwarm")

	fullVolumeName := verification.GetFullVolumeName(s.master, s.volumeName)
	opts := "--replicas 1 --mount type=volume,source=" + fullVolumeName + ",target=/vol,volume-driver=vsphere busybox tail -f /dev/null"
	out, err := dockercli.CreateService(s.master, s.serviceName, opts)
	c.Assert(err, IsNil, Commentf(out))

	status := verification.VerifyDockerService(s.master, s.serviceName, 1)
	c.Assert(status, Equals, true, Commentf("Service %s is not running", s.serviceName))

	status = verification.VerifyDockerContainer(s.dockerHosts, s.serviceName, 1)
	c.Assert(status, Equals, true, Commentf("Container %s is not running", s.serviceName))

	status = verification.VerifyAttachedStatus(s.volumeName, s.master, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	out, err = dockercli.ScaleService(s.master, s.serviceName, 2)
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyDockerService(s.master, s.serviceName, 2)
	c.Assert(status, Equals, true, Commentf("Service %s is not running", s.serviceName))

	status = verification.VerifyDockerContainer(s.dockerHosts, s.serviceName, 2)
	c.Assert(status, Equals, true, Commentf("Container %s is not running", s.serviceName))

	containerName, err = verification.GetContainerName(s.master, s.serviceName+".1")
	c.Assert(err, IsNil, Commentf(out))
	out, err = dockercli.StopService(s.master, containerName)
	c.Assert(err, IsNil, Commentf(out))

	status = verification.VerifyDockerService(s.master, s.serviceName, 2)
	c.Assert(status, Equals, true, Commentf("Service %s is not running", s.serviceName))

	status = verification.VerifyDockerContainer(s.dockerHosts, s.serviceName, 2)
	c.Assert(status, Equals, true, Commentf("Container %s is not running", s.serviceName))

	status = verification.VerifyAttachedStatus(s.volumeName, s.master, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is not attached", s.volumeName))

	out, err = dockercli.DeleteVolume(s.master, s.volumeName)
	c.Assert(err, NotNil, Commentf("Expected error does not happen"))

	out, err = dockercli.RemoveService(s.master, s.serviceName)
	c.Assert(err, IsNil, Commentf(out))

	out, err = dockercli.ListService(s.master, s.serviceName)
	c.Assert(err, NotNil, Commentf("Expected error does not happen"))

	status = verification.VerifyDetachedStatus(s.volumeName, s.master, s.esxName)
	c.Assert(status, Equals, true, Commentf("Volume %s is still attached", s.volumeName))

	log.Printf("END: swarm_test.TestDockerSwarm")
}
