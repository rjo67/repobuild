package repobuild

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// the ModelCli takes commands from the terminal and sends them over the out channel for processing.
// Replies from the model processor are received via the in channel.
func ModelCli(cliCommunication *CliCommunication, wg *sync.WaitGroup) {
	reader := bufio.NewReader(os.Stdin)
	gotStopFromModelProcessor := false
	for running := true; running && !gotStopFromModelProcessor; {
		select {
		case <-cliCommunication.StopChan:
			gotStopFromModelProcessor = true
			fmt.Println("got stop")
		default:
		}
		// TODO rewrite this to get the stdin via a channel - nonblocking -
		//      then can close things down more smoothly
		if !gotStopFromModelProcessor {
			cliCommunication.FromCli <- InChannelObject{cmd: "status"}
			output := <-cliCommunication.ToCli
			fmt.Printf("\n\n%s\n", output.description)
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
					cliCommunication.FromCli <- InChannelObject{cmd: "findRunnableNodes"}
					output := <-cliCommunication.ToCli
					nodeNames, nodeDesc := output.nodeNames, output.nodeDesc
					if len(nodeNames) == 0 {
						fmt.Println("No nodes can be started")
					} else {
						fmt.Printf("Following nodes can be started: %s\n", nodeNames)
					}
					fmt.Printf("RUNNABLE:\n%s\nBLOCKED:\n%s\n", nodeDesc[0], nodeDesc[1])
				case "S":
					split := strings.Split(strings.TrimSpace(cmdStr), " ")
					if len(split) != 3 { // should be 3: "S <node> state"
						fmt.Println("invalid format, must be <nodename> <status> separated by blanks")
					} else {
						// input checking done on model side
						cliCommunication.FromCli <- InChannelObject{cmd: "set", data: fmt.Sprintf("%s %s", split[1], split[2])}
						output := <-cliCommunication.ToCli
						fmt.Printf("%s\n", output.description)
					}
				default:
					fmt.Println("unrecognised command. Use H for help")
				}
			}
		}
	}
	if !gotStopFromModelProcessor {
		cliCommunication.FromCli <- InChannelObject{cmd: "quit"}
		close(cliCommunication.FromCli)
	}
	wg.Done()
}
