package cli

// originally from: https://gist.github.com/miguelmota/301340db93de42b537df5588c1380863

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rjo67/repobuild"
)

// Server
type Server struct {
	ip   net.IP
	port int
}

// Client
type Client struct {
	conn             net.Conn
	cliCommunication repobuild.CliCommunication
}

// New creates a new server instance
func New(ip net.IP, port int) *Server {
	return &Server{
		ip:   ip,
		port: port,
	}
}

const TIMEOUT = 1 * time.Second

// Run starts a simple (telnet) server to take commands from the terminal and communicate via channels with the ModelProcessor.
func (server *Server) Run(cliCommunication *repobuild.CliCommunication, wg *sync.WaitGroup) {

	// using TCP Listener in order to be able to call setDeadline

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: server.ip, Port: server.port})
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Printf("cli running on %s:%d\n", server.ip, server.port)

	for gotStopFromModelProcessor := false; !gotStopFromModelProcessor; {
		select {
		case <-cliCommunication.StopChan:
			gotStopFromModelProcessor = true
		default:
		}
		if !gotStopFromModelProcessor {
			listener.SetDeadline(time.Now().Add(TIMEOUT))
			conn, err := listener.Accept()
			timeout := false
			if err != nil {
				if err, ok := err.(*net.OpError); ok && err.Timeout() {
					// it was a timeout -- ignore
					timeout = true
				} else {
					log.Fatal(err)
				}
			}
			if !timeout {
				client := &Client{
					conn:             conn,
					cliCommunication: *cliCommunication,
				}
				go client.handleRequest()
			}
		}
	}
	fmt.Println("cli_server: exiting")
	wg.Done()
}

const (
	NEWLINE = "\r\n"
)

func (client *Client) handleRequest() {
	client.conn.Write([]byte(processNewlines(fmt.Sprintln("hello"))))
	reader := bufio.NewReader(client.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			client.conn.Close()
			return
		}
		message = strings.Replace(message, "\r\n", "", -1)
		message = strings.Replace(message, "\n", "", -1)
		if len(message) > 0 {
			switch strings.ToUpper(message)[0:1] {
			case "H":
				client.conn.Write([]byte(processNewlines(fmt.Sprintln("Q - quit\n" +
					"H - help\n" +
					"? - get status from model processor\n" +
					"P - process (shows the next build step)\n" +
					"Q - closes this connection\n" +
					"S - set node status (format: <nodename> <status>, e.g. S <node> 1)\n" +
					"X - exit model processor and closes connection"))))
			case "Q":
				client.conn.Write([]byte(processNewlines(fmt.Sprintln("bye"))))
				client.conn.Close()
				return
			case "X":
				client.cliCommunication.FromCli <- repobuild.InChannelObject{Cmd: repobuild.CLI_CMD_EXIT_PROCESSOR}
				output := <-client.cliCommunication.ToCli
				client.conn.Write([]byte(processNewlines(fmt.Sprintf("%s\n", output.Description))))
				client.conn.Write([]byte(processNewlines(fmt.Sprintln("bye"))))
				client.conn.Close()
				return
			case "?":
				client.cliCommunication.FromCli <- repobuild.InChannelObject{Cmd: repobuild.CLI_CMD_STATUS}
				output := <-client.cliCommunication.ToCli
				client.conn.Write([]byte(processNewlines(fmt.Sprintf("%s\n", output.Description))))
			case "P":
				client.cliCommunication.FromCli <- repobuild.InChannelObject{Cmd: repobuild.CLI_CMD_FIND_RUNNABLE_NODES}
				output := <-client.cliCommunication.ToCli
				nodeNames, nodeDesc := output.NodeNames, output.NodeDesc
				if len(nodeNames) == 0 {
					client.conn.Write([]byte(processNewlines(fmt.Sprintln("No nodes can be started"))))
				} else {
					client.conn.Write([]byte(processNewlines(fmt.Sprintf("Following nodes can be started: %s\n", nodeNames))))
				}
				client.conn.Write([]byte(processNewlines(fmt.Sprintf("RUNNABLE:\n%s\nBLOCKED:\n%s\n", nodeDesc[0], nodeDesc[1]))))
			case "S":
				fields := strings.Fields(strings.TrimSpace(message))
				if len(fields) != 3 { // should be 3: "S <node> state"
					client.conn.Write([]byte(processNewlines(fmt.Sprintln("Invalid format, must be <nodename> <status> separated by blanks"))))
				} else {
					// input checking done on model side
					client.cliCommunication.FromCli <- repobuild.InChannelObject{Cmd: repobuild.CLI_CMD_SET, Data: fmt.Sprintf("%s %s", fields[1], fields[2])}
					output := <-client.cliCommunication.ToCli
					client.conn.Write([]byte(processNewlines(fmt.Sprintf("%s\n", output.Description))))
				}
			default:
				client.conn.Write([]byte(processNewlines(fmt.Sprintln("unrecognised command, use H for help"))))
			}
		}
	}
}

// processNewLines replaces \n in the string with the platform-specific EOL. Necessary for telnet clients
func processNewlines(str string) string {
	return strings.ReplaceAll(str, "\n", NEWLINE)
}
