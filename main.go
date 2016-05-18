/**
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/gogo/protobuf/proto"

	log "github.com/golang/glog"
	"github.com/mesos/mesos-go/mesosproto"
	"github.com/mesos/mesos-go/mesosutil"
	mesos_sched "github.com/mesos/mesos-go/scheduler"
	"github.com/mesosphere/mesos-framework-tutorial/server"
	"github.com/mesosphere/mesos-framework-tutorial/utils"
	"github.com/mesosphere/mesos-framework-tutorial/framework_scheduler"
)

const (
	CPUS_PER_TASK       = 1
	MEM_PER_TASK        = 128
	defaultArtifactPort = 12345
)

var (
	address      = flag.String("address", "127.0.0.1", "Binding address for artifact server")
	artifactPort = flag.Int("artifactPort", defaultArtifactPort, "Binding port for artifact server")
	master       = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	executorPath = flag.String("executor", "./example_executor", "Path to test executor")
	taskCount    = flag.String("task-count", "5", "Total task count to run.")
)

func init() {
	flag.Parse()
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
		os.Exit(-1)
	}
}

func main() {
	// Start HTTP server that statically hosts the executor binary file
	// returns a "http://[host]:[port]/[executor_file_name]"
	uri := server.LaunchExecutorArtifactServer(*address, *artifactPort, *executorPath)

	// ./[executor_file_name]
	executorCmd := utils.GetExecutorCmd(*executorPath)

	// uri is "http://[host]:[port]/[executor_file_name]", executorCmd
	// is ./[executor_file_name]
	//Gets a ExecutorInfo
	exec := prepareExecutorInfo(uri, executorCmd)

	// Scheduler
	httpServerAddress := fmt.Sprintf("http://%s:%d", *address, *artifactPort)
	log.Infof("ServerAdress: %v\n", httpServerAddress)
	scheduler, err := framework_scheduler.NewExampleScheduler(exec, CPUS_PER_TASK, MEM_PER_TASK, httpServerAddress)
	if err != nil {
		log.Fatalf("Failed to create scheduler with error: %v\n", err)
		os.Exit(-2)
	}

	// Framework
	frameworkInfo := &mesosproto.FrameworkInfo{
		User: proto.String(""), // Mesos-go will fill in user.
		Name: proto.String("Stratio Test Framework (Go)"),
	}

	//Returns a net.IP object from a string address
	ip := utils.ParseIP(*address)

	// Scheduler Driver
	config := mesos_sched.DriverConfig{
		Scheduler:      scheduler,
		Framework:      frameworkInfo,
		Master:         *master,
		Credential:     (*mesosproto.Credential)(nil),
		BindingAddress: ip,
	}

	driver, err := mesos_sched.NewMesosSchedulerDriver(config)

	if err != nil {
		log.Fatalf("Unable to create a SchedulerDriver: %v\n", err.Error())
		os.Exit(-3)
	}

	if stat, err := driver.Run(); err != nil {
		log.Fatalf("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
		os.Exit(-4)
	}
}

// prepareExecutorInfo receives uri like "http://[host]:[port]/[executor_file_name]"
// and a cmd like "./[executor_file_name]" (last element of the uri)
func prepareExecutorInfo(uri string, cmd string) *mesosproto.ExecutorInfo {
	//TODO check what is this. Seems like it creates a CommandInfo_URI to hold
	//TODO the value of http://[host]:[port]/[executor_file_name] and sets it
	//TODO as an executable
	executorUris := []*mesosproto.CommandInfo_URI{
		{
			Value:      &uri,
			Executable: proto.Bool(true),
		},
	}

	// Fills an executor with hardcoded identifiers. The interesting thing is that
	// it also creates a CommandInfo object with Value=./[executor_file_name] and
	// Uris=[executorUris] which simply points to "http://[host]:[port]/[executor_file_name]"
	return &mesosproto.ExecutorInfo{
		ExecutorId: mesosutil.NewExecutorID("Stratio"),
		Name:       proto.String("Stratio Executor (Go)"),
		Source:     proto.String("go_test"),
		Command: &mesosproto.CommandInfo{
			Value: proto.String(cmd),
			Uris:  executorUris,
		},
	}
}

func getExecutorCmd(path string) string {
	return "." + server.GetHttpPath(path)
}

func parseIP(address string) net.IP {
	addr, err := net.LookupIP(address)
	if err != nil {
		log.Fatal(err)
	}
	if len(addr) < 1 {
		log.Fatalf("failed to parse IP from address '%v'", address)
	}
	return addr[0]
}
