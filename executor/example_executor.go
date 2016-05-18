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
	"encoding/json"
	"flag"
	"fmt"

	mesos_executor "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	payload "github.com/mesosphere/mesos-framework-tutorial/payload"
)

type exampleExecutor struct {
	tasksLaunched int
}

func (exec *exampleExecutor) LaunchTask(driver mesos_executor.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	fmt.Printf("Launching task id: %s - %v with data [%#x]\n", taskInfo.GetTaskId().Value, taskInfo.GetName(), taskInfo.Data)

	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}

	exec.tasksLaunched++
	fmt.Println("Total tasks launched ", exec.tasksLaunched)

	// Decode payload
	var decodedPayload payload.TaskPayload
	err = json.Unmarshal(taskInfo.Data, &decodedPayload)
	fmt.Printf("Payload Out: %v\n", decodedPayload)

	// Download image
	fileName, err := downloadImage(string(decodedPayload.FileName))
	if err != nil {
		fmt.Printf("Failed to download image '%v' with error: %v\n", fileName, err)
		return
	}
	fmt.Printf("Downloaded image: %v\n", fileName)

	// Process image
	fmt.Printf("Processing image: %v\n", fileName)
	outFile, err := procImage(fileName)
	if err != nil {
		fmt.Printf("Failed to process image with error: %v\n", err)
		return
	}

	// Upload image
	fmt.Printf("Uploading image: %v\n", outFile)
	if err = uploadImage(decodedPayload.HttpServerAddress, outFile); err != nil {
		fmt.Printf("Failed to upload image with error: %v\n", err)
		return
	} else {
		fmt.Printf("Uploaded image: %v\n", outFile)
	}

	// Finish task
	fmt.Println("Finishing task", taskInfo.GetName())
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		fmt.Println("Got error", err)
		return
	}

	fmt.Println("Task finished", taskInfo.GetName())
}

//init parses the flags passed by the command line
func init() {
	flag.Parse()
}

func main() {
	fmt.Println("Starting Example Executor (Go)")

	dconfig := mesos_executor.DriverConfig{
		Executor: &exampleExecutor{tasksLaunched: 0},
	}
	driver, err := mesos_executor.NewMesosExecutorDriver(dconfig)

	if err != nil {
		fmt.Println("Unable to create a ExecutorDriver ", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		fmt.Println("Got error:", err)
		return
	}
	fmt.Println("Executor process has started and running.")
	driver.Join()
}

func (exec *exampleExecutor) Registered(driver mesos_executor.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Reregistered(driver mesos_executor.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Disconnected(mesos_executor.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (exec *exampleExecutor) KillTask(mesos_executor.ExecutorDriver, *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (exec *exampleExecutor) FrameworkMessage(driver mesos_executor.ExecutorDriver, msg string) {
	fmt.Println("Got framework message: ", msg)
}

func (exec *exampleExecutor) Shutdown(mesos_executor.ExecutorDriver) {
	fmt.Println("Shutting down the executor")
}

func (exec *exampleExecutor) Error(driver mesos_executor.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}
