package server

import "testing"

func TestGetHttpPath(t *testing.T) {
	res := GetHttpPath("/home/mcastro/go/src/github.com/mesosphere/mesos-framework-tutorial/executor/example_executor")
	if res[0] != '/' {
		t.Error("First character wasn't a '/'")
	}

	t.Log(res)
}
