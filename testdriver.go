package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func testDriver(model Model) {
	reader := bufio.NewReader(os.Stdin)
	for running := true; running; {
		fmt.Printf("%s\n", model.status(true))
		fmt.Println("\nSet node status, Process, Quit, Help")
		cmdStr, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("error: %v\n", err)
		} else {
			switch strings.ToUpper(cmdStr)[0:1] {
			case "H":
				fmt.Println(`
Q - quit
H - help
S - set node status (format: <nodename> <status>, e.g. S <node> 1)
P - process (shows the next build step)
					`)
			case "Q":
				running = false
			case "P":
				nodes, desc := model.findRunnableNodes()
				if len(nodes[0]) == 0 {
					fmt.Println("No nodes can be started")
				} else {
					nodeNames := ""
					for _, node := range nodes[0] {
						nodeNames += node.Name + " "
					}
					fmt.Printf("Following nodes can be started: %s\n", nodeNames)
				}
				fmt.Printf("** Detailed info:\nRUNNABLE: %s\nBLOCKED: %s\n**\n", desc[0], desc[1])
			case "S":
				split := strings.Split(strings.TrimSpace(cmdStr), " ")
				if len(split) != 3 { // should be 3: "S <node> state"
					fmt.Println("invalid format, must be <nodename> <status> separated by blanks")
				} else {
					nodeName := split[1]
					state, err := strconv.Atoi(split[2])
					if err != nil || state < 0 || state > ERROR {
						fmt.Printf("invalid value for state (must be 0..%d)\n", ERROR)
					} else {
						if node, present := model.Nodes[nodeName]; present {
							fmt.Printf("Setting node %s to state %d\n", nodeName, state)
							node.Status = state
						} else {
							fmt.Printf("Node '%s' not defined\n", nodeName)
						}
					}
				}
			}
		}
	}
}
