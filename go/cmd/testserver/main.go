package main

import (
	"fmt"

	"github.com/rjo67/repobuild/testserver"
)

// A simple tcp server to mimic a build server.
// Can receive a command to initiate a 'build' of a project and spawns a process to do so
// Can be interrogated as to the status of the project build.
func main() {
	host := "localhost"
	port := "3333"
	buildServer := testserver.New(&testserver.Config{
		Host: host,
		Port: port,
	})
	fmt.Printf("testserver running: %v\n", *buildServer)
	buildServer.Run()
}
