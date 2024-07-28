package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	var filename string
	flag.StringVar(&filename, "f", "", "yaml file")
	flag.Parse()

	// check input parameters
	if len(filename) <= 0 {
		fmt.Println("must specify filename")
		os.Exit(1)
	}

	// run program

	if err := run(filename); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(filename string) error {
	inputData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	yamlModel, err := loadYamlModel(inputData)
	if err != nil {
		return err
	}
	model, err := createModel(yamlModel)
	if err != nil {
		return err
	}

	testDriver(model)

	return nil
}

func testDriver(model Model) {
	reader := bufio.NewReader(os.Stdin)
	for running := true; running; {
		fmt.Printf("%s\n", model.status())
		fmt.Println("Set node status, Process, Quit, Help")
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
				nodes := model.findRunnableNodes()
				fmt.Printf("got nodes: %v\n", nodes)
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

func stringAddr(obj any) string {
	return fmt.Sprintf("%p_%s_", &obj, obj)
}
