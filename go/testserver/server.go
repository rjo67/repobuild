package testserver

// originally from: https://gist.github.com/miguelmota/301340db93de42b537df5588c1380863

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
)

// Server
type Server struct {
	host string
	port string
}

// Client
type Client struct {
	conn      net.Conn
	processes *map[string]int
}

// Config for the server
type Config struct {
	Host string
	Port string
}

// New creates a new server instance
func New(config *Config) *Server {
	return &Server{
		host: config.Host,
		port: config.Port,
	}
}

type Command struct {
	cmd  string
	args []string
}

// Status values
const (
	WAITING  = 0
	RUNNING  = iota
	FINISHED = iota
	ERROR    = iota
)

// Run
func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	// stores processes and their status
	// TODO should make this thread-safe
	processMap := make(map[string]int)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		client := &Client{
			conn:      conn,
			processes: &processMap,
		}
		go client.handleRequest()
	}
}

func (client *Client) handleRequest() {
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
			fmt.Printf("got message '%s'\n", message)
			cmd, err := parseMessage(message)
			if err != nil {
				client.conn.Write([]byte(fmt.Sprintf("%v\n", err)))
			} else {
				switch cmd.cmd {
				case "START":
					if len(cmd.args) >= 1 {
						processName := cmd.args[0]
						if currentState, present := (*client.processes)[processName]; present {
							client.conn.Write([]byte(fmt.Sprintf("process: %s already in state: %d\n", processName, currentState)))
						} else {
							(*client.processes)[processName] = RUNNING
							client.conn.Write([]byte("ok\n"))
						}
					} else {
						client.conn.Write([]byte(fmt.Sprintln("? process name not specified")))
					}
				case "FINISH", "ERROR":
					if len(cmd.args) >= 1 {
						processName := cmd.args[0]
						currentState, present := (*client.processes)[processName]
						if !present || currentState != RUNNING {
							client.conn.Write([]byte(fmt.Sprintf("process: %s is not running\n", processName)))
						} else {
							(*client.processes)[processName] = convertToState(cmd.cmd)
							client.conn.Write([]byte("ok\n"))
						}
					} else {
						client.conn.Write([]byte(fmt.Sprintln("? process name not specified")))
					}
				case "STATUS":
					if len(cmd.args) >= 1 {
						processName := cmd.args[0]
						if _, present := (*client.processes)[processName]; present {
							client.conn.Write([]byte(fmt.Sprintf("%s,%d\n", processName, (*client.processes)[processName])))
						} else {
							client.conn.Write([]byte(fmt.Sprintf("? process: %s\n", processName)))
						}
					} else {
						client.conn.Write([]byte(fmt.Sprintf("%v\n", *client.processes)))
					}
				case "QUIT":
					client.conn.Close()
					return
				default:
					client.conn.Write([]byte(fmt.Sprintf("unrecognised command '%s', valid commands are: %s\n", cmd.cmd, "STATUS START FINISH ERROR QUIT")))
				}
			}
		}
	}
}

func parseMessage(message string) (cmd Command, err error) {
	split := strings.Split(strings.TrimSpace(message), " ")
	if len(split) == 0 {
		return Command{}, fmt.Errorf("Invalid syntax, received:" + message)
	}
	cmd = Command{cmd: strings.ToUpper(split[0])}
	if len(split) > 1 {
		cmd.args = split[1:]
	}
	return cmd, nil
}

func convertToState(str string) int {
	switch str {
	case "FINISH":
		return FINISHED
	case "ERROR":
		return ERROR
	default:
		return -1
	}
}
