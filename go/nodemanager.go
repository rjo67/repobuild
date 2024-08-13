package repobuild

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// NodeManager communicates with the ModelProcessor (via commands on cmdChannel) and returns the state of finished Projects on cmdReplyChannel
func NodeManager(cmdChannel <-chan Command, cmdReplyChannel chan<- Command) {
	infoChannel := make(chan string) // receives FINISHED info from processes
	for stop := false; !stop; {
		select {
		case cmdInput := <-cmdChannel:
			switch cmdInput.cmd {
			case CMD_STOP:
				stop = true
			case CMD_STARTNODE:
				startNode(cmdInput.data, infoChannel)
			}
		case infoInput := <-infoChannel:
			infoFields := strings.Fields(infoInput)
			switch infoFields[0] {
			case CMDREPLY_FINISHED:
				cmdReplyChannel <- Command{cmd: CMDREPLY_FINISHED, data: infoFields[1]}
			case CMDREPLY_ERROR:
				cmdReplyChannel <- Command{cmd: CMDREPLY_ERROR, data: infoFields[1]}
			default:
				cmdReplyChannel <- Command{cmd: CMDREPLY_UNKNOWN, data: infoInput}
			}
		}
	}
}

func startNode(name string, reply chan<- string) {
	go TestNode(name, reply)
}

// TestNode sleeps for a random amount of time and then returns FINISHED on the channel
func TestNode(name string, reply chan<- string) {
	sleep := rand.IntN(10) + 1
	fmt.Printf("Node %s starting (sleep=%d)\n", name, sleep)
	time.Sleep(time.Duration(sleep) * time.Second)
	reply <- fmt.Sprintf("%s %s", CMDREPLY_FINISHED, name)
}
