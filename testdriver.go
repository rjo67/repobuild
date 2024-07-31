package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// the testDriver receives commands from the terminal and sends them over the out channel for processing.
// Replies from the model processor are received via the in channel.
func testDriver(fromProcessor chan OutChannelObject, toProcessor chan InChannelObject, wg sync.WaitGroup) {
	reader := bufio.NewReader(os.Stdin)
	for running := true; running; {
		toProcessor <- InChannelObject{cmd: "status"}
		output := <-fromProcessor
		fmt.Printf("%s\n", output.description)
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
				close(toProcessor)
				wg.Done()
			case "P":
				toProcessor <- InChannelObject{cmd: "findRunnableNodes"}
				output := <-fromProcessor
				nodeNames, nodeDesc := output.nodeNames, output.nodeDesc
				if len(nodeNames[0]) == 0 {
					fmt.Println("No nodes can be started")
				} else {
					fmt.Printf("Following nodes can be started: %s\n", nodeNames)
				}
				fmt.Printf("** Detailed info:\nRUNNABLE: %s\nBLOCKED: %s\n**\n", nodeDesc[0], nodeDesc[1])
			case "S":
				split := strings.Split(strings.TrimSpace(cmdStr), " ")
				if len(split) != 3 { // should be 3: "S <node> state"
					fmt.Println("invalid format, must be <nodename> <status> separated by blanks")
				} else {
					// input checking done on model side
					toProcessor <- InChannelObject{cmd: "set", data: fmt.Sprintf("%s %s", split[1], split[2])}
					output := <-fromProcessor
					fmt.Printf("%s\n", output.description)
				}
			}
		}
	}
}
