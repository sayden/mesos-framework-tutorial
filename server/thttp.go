package server

import (
	"net/http"
	"strings"

	"fmt"
)

//ServeExecutorArtifact is called from main function
func LaunchExecutorArtifactServer(address string, port int, filePath string) string {
	httpPath := GetHttpPath(filePath)

	serverURI := fmt.Sprintf("%s:%d", address, port)
	hostURI := fmt.Sprintf("http://%s%s", serverURI, httpPath)

	http.HandleFunc(httpPath, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, httpPath)
	})

	go http.ListenAndServe(address, nil)

	return hostURI
}

// GetHttpPath returns from a "http://foobar:5000/<last>" a "/<last>"
func GetHttpPath(path string) string {
	// Create base path (http://foobar:5000/<base>)
	pathSplit := strings.Split(path, "/")
	var base string
	if len(pathSplit) > 0 {
		base = pathSplit[len(pathSplit)-1]
	} else {
		base = path
	}

	return "/" + base
}
