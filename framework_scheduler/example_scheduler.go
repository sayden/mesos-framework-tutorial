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

package framework_scheduler

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strconv"

	"os"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
	payload "github.com/mesosphere/mesos-framework-tutorial/payload"
)

type ExampleScheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
	totalTasks    int
	images        []string
	cpuPerTask    float64
	memPerTask    float64
	serverAddress string
}

func NewExampleScheduler(exec *mesos.ExecutorInfo, cpuPerTask float64, memPerTask float64, serverAddress string) (*ExampleScheduler, error) {
	images, err := readLines("images")
	if err != nil {
		log.Errorf("Failed to read image list with error: %v\n", err)
		return nil, err
	}

	return &ExampleScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
		totalTasks:    5,
		images:        images,
		cpuPerTask:    cpuPerTask,
		memPerTask:    memPerTask,
		serverAddress: serverAddress,
	}, nil
}

func (sched *ExampleScheduler) Registered(driver sched.SchedulerDriver, frameworkId *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Scheduler Re-Registered with Master ", masterInfo)
}

func (sched *ExampleScheduler) Disconnected(sched.SchedulerDriver) {
	log.Infoln("Scheduler Disconnected")
}

func (sched *ExampleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	log.Infoln("ResourceOffers called")
	logOffers(offers)

	if sched.tasksLaunched >= sched.totalTasks {
		return
	}

	for _, offer := range offers {
		remainingCpus := getOfferCpu(offer)
		remainingMems := getOfferMem(offer)

		fmt.Printf("CPUs: %d, Mem: %d", remainingCpus, remainingMems)

		var tasks []*mesos.TaskInfo
		for sched.cpuPerTask <= remainingCpus &&
			sched.memPerTask <= remainingMems &&
			sched.tasksLaunched < sched.totalTasks {

			log.Infof("Processing image %v of %v\n", sched.tasksLaunched, sched.totalTasks)

			// Build payload
			fileName := sched.images[sched.tasksLaunched]
			payld := payload.TaskPayload{fileName, sched.serverAddress}
			log.Infof("Payload In: %v\n", payld)

			b, err := json.Marshal(payld)
			if err != nil {
				log.Errorf("Failed to marshal payload with error: %v\n", err)
				return
			}

			sched.tasksLaunched++

			taskId := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}

			task := &mesos.TaskInfo{
				Name:     proto.String("go-task-" + taskId.GetValue()),
				TaskId:   taskId,
				SlaveId:  offer.SlaveId,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", sched.cpuPerTask),
					util.NewScalarResource("mem", sched.memPerTask),
				},
				Data: b,
			}
			log.Infof("Prepared task: %s with offer %s for launch\n", task.GetName(), offer.Id.GetValue())

			tasks = append(tasks, task)
			remainingCpus -= sched.cpuPerTask
			remainingMems -= sched.memPerTask
		}
		log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

func (sched *ExampleScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())

	if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.tasksFinished++
		log.Infof("%v of %v tasks finished.", sched.tasksFinished, sched.totalTasks)
	}

	if sched.tasksFinished >= sched.totalTasks {
		log.Infoln("Total tasks completed, stopping framework.")
		driver.Stop(false)
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Infoln(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

func (sched *ExampleScheduler) OfferRescinded(s sched.SchedulerDriver, id *mesos.OfferID) {
	log.Infof("Offer '%v' rescinded.\n", *id)
}

func (sched *ExampleScheduler) FrameworkMessage(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, msg string) {
	log.Infof("Received framework message from executor '%v' on slave '%v': %s.\n", *exId, *slvId, msg)
}

func (sched *ExampleScheduler) SlaveLost(s sched.SchedulerDriver, id *mesos.SlaveID) {
	log.Infof("Slave '%v' lost.\n", *id)
}

func (sched *ExampleScheduler) ExecutorLost(s sched.SchedulerDriver, exId *mesos.ExecutorID, slvId *mesos.SlaveID, i int) {
	log.Infof("Executor '%v' lost on slave '%v' with exit code: %v.\n", *exId, *slvId, i)
}

func (sched *ExampleScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}

func getOfferScalar(offer *mesos.Offer, name string) float64 {
	resources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
		return res.GetName() == name
	})

	value := 0.0
	for _, res := range resources {
		value += res.GetScalar().GetValue()
	}

	return value
}

func getOfferCpu(offer *mesos.Offer) float64 {
	return getOfferScalar(offer, "cpus")
}

func getOfferMem(offer *mesos.Offer) float64 {
	return getOfferScalar(offer, "mem")
}

func logOffers(offers []*mesos.Offer) {
	for _, offer := range offers {
		log.Infof("Received Offer <%v> with cpus=%v mem=%v", offer.Id.GetValue(), getOfferCpu(offer), getOfferMem(offer))
	}
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
