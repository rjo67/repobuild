package repobuild

import (
	"fmt"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Status values
const (
	WAITING = iota
	RUNNING
	FINISHED
	ERROR
	IGNORED
	numberOfStatusValues
)

var statusString = map[int]string{
	WAITING:  "waiting",
	RUNNING:  "running",
	FINISHED: "finished",
	ERROR:    "in error",
	IGNORED:  "ignored",
}

// SLEEP_WAIT_DURATION is how long the ModelProcessor waits before updating its state
const SLEEP_WAIT_DURATION = 500 * time.Millisecond

func statusToString(status int) string {
	if status < numberOfStatusValues && status >= 0 {
		return statusString[status]
	} else {
		return fmt.Sprintf("status not recognised: %d", status)
	}
}
func hasFinishedStatus(status int) bool {
	return status == FINISHED || status == IGNORED
}

// Model stores the Nodes
type Model struct {
	Nodes map[string]*Node
}

// NewModel creates a new Model from the YamlModel
func NewModel(yamlModel YamlModel) (*Model, error) {
	modelMap := make(map[string]*Node)
	var err error
	// first create all Node objects in map
	for _, yamlNode := range yamlModel.Nodes {
		nodeName := yamlNode.Name
		if _, present := modelMap[nodeName]; present {
			return &Model{}, fmt.Errorf("node '%s' defined multiple times", nodeName)
		}
		nodeStatus := WAITING
		if yamlNode.Ignore {
			nodeStatus = IGNORED
		}
		node := Node{Name: nodeName, Script: yamlNode.Script, Status: nodeStatus}
		modelMap[nodeName] = &node
	}
	// process dependencies
	for _, yamlNode := range yamlModel.Nodes {
		nodeName := yamlNode.Name
		if ancestors, err := _processAncestors(modelMap, yamlNode); err != nil {
			return &Model{}, err
		} else {
			node := modelMap[nodeName]
			node.Ancestors = ancestors
		}
	}
	return &Model{Nodes: modelMap}, err
}

// nodesWithStatus returns an array of slices of nodes indexed on the required status (indexed by 'Status')
func (m Model) nodesWithStatus() [numberOfStatusValues][]*Node {
	var nodes [numberOfStatusValues][]*Node
	for i := 0; i < numberOfStatusValues; i++ {
		nodes[i] = make([]*Node, 0)
	}
	for _, node := range m.Nodes {
		nodes[node.Status] = append(nodes[node.Status], node)
	}
	return nodes
}

// nodesInEndStatus returns how many nodes are in an 'end' status (finished, error, ignored)
func (m Model) nodesInEndStatus() int {
	allNodes := m.nodesWithStatus()
	return len(allNodes[FINISHED]) + len(allNodes[ERROR]) + len(allNodes[IGNORED])
}

// SetIgnored sets the 'ignored' flag on the nodes in the eparated list 'ignored'
// Returns the number of nodes processed
func (m Model) SetIgnored(ignored string) (int, error) {
	nodeNames := strings.Split(ignored, ",")
	for _, name := range nodeNames {
		if node, present := m.Nodes[name]; present {
			node.Status = IGNORED
		} else {
			return -1, fmt.Errorf("unknown node '%s' in ignore list", name)
		}
	}
	return len(nodeNames), nil
}

// getNodeNames returns the sorted names of the nodes in the given parameter
func getNodeNames(nodes []*Node) []string {
	var result []string
	for _, node := range nodes {
		result = append(result, node.Name)
	}
	slices.Sort(result)
	return result
}

// status returns an overview of all nodes
func (m Model) status(verbose bool) string {
	allNodes := m.nodesWithStatus()
	var nbrNodes [numberOfStatusValues]int
	var nodeNames [numberOfStatusValues][]string
	totalNbrNodes := 0
	for i := 0; i <= IGNORED; i++ {
		nbrNodes[i] = len(allNodes[i])
		totalNbrNodes += nbrNodes[i]
		nodeNames[i] = getNodeNames(allNodes[i])
	}
	// 'error' and 'ignored' special cases, since don't occur so frequently
	var errStr = ""
	if nbrNodes[ERROR] != 0 {
		if verbose {
			errStr = fmt.Sprintf("%d in error: %s\n", nbrNodes[ERROR], nodeNames[ERROR])
		} else {
			errStr = fmt.Sprintf("%d in error", nbrNodes[ERROR])
		}
	}
	var ignoredStr = ""
	if nbrNodes[IGNORED] != 0 {
		if verbose {
			ignoredStr = fmt.Sprintf("%d ignored: %s\n", nbrNodes[IGNORED], nodeNames[IGNORED])
		} else {
			ignoredStr = fmt.Sprintf("%d ignored, ", nbrNodes[IGNORED])
		}
	}
	// adjust formatting: if the 'ignoredStr' is empty, then need a comma after 'errStr' (if not empty)
	if ignoredStr == "" {
		if errStr != "" {
			errStr += ", "
		}
	}
	var percents [numberOfStatusValues]float32
	totalNbrNodesExcludingIgnored := totalNbrNodes - nbrNodes[IGNORED]
	percents[WAITING] = float32((nbrNodes[WAITING] * 100.0) / totalNbrNodesExcludingIgnored)
	percents[RUNNING] = float32((nbrNodes[RUNNING] * 100.0) / totalNbrNodesExcludingIgnored)
	percents[FINISHED] = float32((nbrNodes[FINISHED] * 100.0) / totalNbrNodesExcludingIgnored)
	padding := int(math.Round(math.Log10(float64(totalNbrNodes)))) + 1
	if verbose {
		return fmt.Sprintf("%*d waiting  (%.0f%%): %s\n%*d running  (%.0f%%): %s\n%*d finished (%.0f%%): %s\n%s%sTOTAL: %d",
			padding, nbrNodes[WAITING], percents[WAITING], nodeNames[WAITING],
			padding, nbrNodes[RUNNING], percents[RUNNING], nodeNames[RUNNING],
			padding, nbrNodes[FINISHED], percents[FINISHED], nodeNames[FINISHED],
			errStr, ignoredStr, totalNbrNodes)
	} else {
		return fmt.Sprintf("%d waiting, %d running, %d finished (%.0f%%), %s%sTOTAL: %d", nbrNodes[WAITING], nbrNodes[RUNNING],
			nbrNodes[FINISHED], percents[FINISHED], errStr, ignoredStr, totalNbrNodes)
	}
}

// findRunnableNodes returns a 2 dimensional array of nodes.
// The first dimension contains the nodes which can be started.
// The second dimension contains those nodes which cannot be started.
// A node is startable if all its ancestors have status=FINISHED.
// The stateDescription array describes why the nodes can or cannot be started.
func (m Model) findRunnableNodes() (nodes [2][]*Node, stateDescription [2][]string) {
	allNodes := m.nodesWithStatus()
	for _, node := range allNodes[WAITING] {
		allAncestorsFinished, ancestorsDesc := node.ancestorsAreFinished()
		var slot int
		if allAncestorsFinished {
			slot = 0
		} else {
			slot = 1
		}
		nodes[slot] = append(nodes[slot], node)
		if len(ancestorsDesc) == 0 {
			ancestorsDesc = "no ancestors"
		} else {
			ancestorsDesc = "ancestors: " + ancestorsDesc
		}
		stateDescription[slot] = append(stateDescription[slot], fmt.Sprintf("%s (%s)\n", node.Name, ancestorsDesc))
	}
	for slot := 0; slot < len(nodes); slot++ {
		sort.Strings(stateDescription[slot])
		sort.Slice(nodes[slot], func(i, j int) bool { return nodes[slot][i].Name < nodes[slot][j].Name })
	}
	return nodes, stateDescription
}

// DetectCycle inspects the model to detect a possible cycle.
func (m Model) DetectCycle() error {
	nameStack := Stack[string]()
	visited := make(map[string]bool)
	recorded := make(map[string]bool)
	for _, node := range m.Nodes {
		if _, alreadyVisited := visited[node.Name]; !alreadyVisited {
			err := _detectCycle(node, &visited, &recorded, nameStack)
			if err != nil {
				return err
			}
		}
		//		nameStack.Pop()
	}
	return nil
}

// _detectCycle returns an error if the model contains a cycle
func _detectCycle(node *Node, visited *map[string]bool, recorded *map[string]bool, nameStack stack[string]) error {
	(*visited)[node.Name] = true
	(*recorded)[node.Name] = true
	nameStack.Push(node.Name)
	for _, ancestor := range node.Ancestors {
		if alreadyVisited := (*visited)[ancestor.Name]; !alreadyVisited {
			err := _detectCycle(ancestor, visited, recorded, nameStack)
			if err != nil {
				return err
			}
			nameStack.Pop()
		} else if alreadyRecorded := (*recorded)[ancestor.Name]; alreadyRecorded {
			nameStack.Push(ancestor.Name) // to improve error message
			return fmt.Errorf("cycle detected %v", nameStack.Elements())
		}
	}
	(*recorded)[node.Name] = false
	return nil
}

type Node struct {
	Name      string
	Script    string
	Status    int
	Ancestors []*Node
	Children  []*Node
}

// ancestorsAreFinished returns whether all the ancestors of this node have state FINISHED.
// The second parameter describes the state of all ancestors.
func (n Node) ancestorsAreFinished() (allAncestorsFinished bool, desc string) {
	allAncestorsFinished = true
	var ancestorStatus [numberOfStatusValues][]string
	for _, node := range n.Ancestors {
		ancestorStatus[node.Status] = append(ancestorStatus[node.Status], node.Name)
		if !hasFinishedStatus(node.Status) {
			allAncestorsFinished = false
		}
	}
	for status := WAITING; status < numberOfStatusValues; status++ {
		if len(ancestorStatus[status]) != 0 {
			if len(desc) != 0 {
				desc += ", "
			}
			sort.Strings(ancestorStatus[status])
			desc += fmt.Sprintf("%s=%s", statusToString(status), ancestorStatus[status])
		}
	}
	return allAncestorsFinished, desc
}

func (n Node) String() string {
	// TODO Script not yet included here
	return fmt.Sprintf("(%s/%s/%s)", n.Name, n.Ancestors, n.Children)
}

// command names for struct Command
const (
	CMD_STARTNODE     = "STARTNODE"
	CMD_STOP          = "STOP"
	CMDREPLY_FINISHED = "FINISHED"
	CMDREPLY_ERROR    = "ERROR"
	CMDREPLY_UNKNOWN  = "UNKNOWN"
)

// commands from CLI
const (
	CLI_CMD_EXIT_PROCESSOR      = "exit"
	CLI_CMD_FIND_RUNNABLE_NODES = "findRunnableNodes"
	CLI_CMD_SET                 = "set"
	CLI_CMD_STATUS              = "status"
)

// Command is the data structure that's passed between ModelProcessor and NodeManager
type Command struct {
	cmd  string // name of command
	data string // data for command
}

// ModelProcessor is the main loop to process the model.
// Will loop for as long as there are non-finished nodes, processing any nodes which can be run.
// If the CLI is in use (useCli), then nothing will happen automatically.
// Simple commands can be sent via the cliCommunication 'in' Channel (results are sent back on the 'out' Channel)
// Statistics will be stored in stats.
func (model *Model) ModelProcessor(useCli bool, cliCommunication *CliCommunication, stats *Statistics, wg *sync.WaitGroup) {
	cmdChannel := make(chan Command)
	cmdReplyChannel := make(chan Command)

	nodeStatusChanged := false
	nodeStartTime := make(map[string]time.Time) // stores start times for nodes
	// add 'ignored' nodes to stats
	for _, ignoredNode := range model.nodesWithStatus()[IGNORED] {
		stats.NodeStats = append(stats.NodeStats, NodeStatistics{Name: ignoredNode.Name, BuildTime: time.Duration(0)})
	}

	if useCli {
		fmt.Println("Waiting for CLI commands....")
	}
	go NodeManager(cmdChannel, cmdReplyChannel)
	runningNodes := 0
	for stop := false; !stop; {
		select {
		// get input from model cli
		case input := <-cliCommunication.FromCli:
			stop = model._processCommand(input, cliCommunication.ToCli)
			//TODO integrate processing of CLI commands with 'pmReply' processing: e.g. in order to have stats
		// get info from NodeManager
		case pmReply := <-cmdReplyChannel:
			switch pmReply.cmd {
			case CMDREPLY_FINISHED, CMDREPLY_ERROR:
				node, present := model.Nodes[pmReply.data]
				if !present {
					fmt.Printf("Unknown node in reply from NodeManager: %v\n", pmReply)
				} else if node.Status != RUNNING {
					fmt.Printf("Invalid status for node %s: %s\n", pmReply.data, statusToString(node.Status))
				} else {
					stateStr := ""
					if pmReply.cmd == CMDREPLY_FINISHED {
						node.Status = FINISHED
					} else {
						node.Status = ERROR
						stateStr = " (state ERROR)"
					}
					fmt.Printf("Node %s finished%s\n", node.Name, stateStr)
					runningNodes--
					if startTime, present := nodeStartTime[node.Name]; present {
						stats.NodeStats = append(stats.NodeStats,
							NodeStatistics{Name: node.Name, BuildTime: time.Since(startTime).Round(time.Second)})
					}
					nodeStatusChanged = true
				}
			default:
				fmt.Printf("Unrecognised message on cmd channel: %v\n", pmReply)
			}
		default:
			// update state of model (have tasks finished? Start new tasks, etc)
			if !useCli {
				runnableNodes, _ := model.findRunnableNodes()
				if len(runnableNodes[0]) == 0 && runningNodes == 0 {
					// deadlock
					stop = true
					fmt.Printf("deadlock")
				}
				for _, node := range runnableNodes[0] {
					nodeStartTime[node.Name] = time.Now()
					cmdChannel <- Command{cmd: CMD_STARTNODE, data: node.Name}
					node.Status = RUNNING
					runningNodes++
					nodeStatusChanged = true
				}
			}
		}
		if nodeStatusChanged {
			time.Sleep(SLEEP_WAIT_DURATION)
			fmt.Printf("\n\n%s\n", model.status(false))
			nodeStatusChanged = false
		}
		// potentially exit the loop, if all finished, but only if CLI not active
		if len(model.Nodes) == model.nodesInEndStatus() {
			if !useCli {
				stop = true
			}
		}
	}

	// stop node manager
	cmdChannel <- Command{cmd: CMD_STOP}

	// tell cli to stop (if present)
	if useCli {
		cliCommunication.ToCli <- OutChannelObject{Description: "model processor terminating"}
		cliCommunication.StopChan <- 1
	}
	wg.Done()
}

// _processCommands processes a CLI command and sends an answer back on outChannel
// Returns true if the calling loop should exit
func (model Model) _processCommand(input InChannelObject, outChannel chan<- OutChannelObject) (quitRequested bool) {
	quitRequested = false
	switch input.Cmd {
	case CLI_CMD_EXIT_PROCESSOR:
		quitRequested = true
	case CLI_CMD_STATUS:
		outChannel <- OutChannelObject{Description: model.status(true)}
	case CLI_CMD_FIND_RUNNABLE_NODES:
		nodes, nodesDesc := model.findRunnableNodes()
		// just return the node names
		var nodeStr []string
		for _, node := range nodes[0] {
			nodeStr = append(nodeStr, node.Name)
		}
		outChannel <- OutChannelObject{NodeNames: nodeStr, NodeDesc: nodesDesc}
	case CLI_CMD_SET:
		split := strings.Split(strings.TrimSpace(input.Data), " ")
		if len(split) != 2 { // should be 3: "<node> <state>"
			outChannel <- OutChannelObject{Description: fmt.Sprintf("invalid format, must be '<node> <state>', got: %s", input.Data)}
		} else {
			nodeName := split[0]
			state, err := strconv.Atoi(split[1])
			if err != nil || state < 0 || state > ERROR {
				outChannel <- OutChannelObject{Description: fmt.Sprintf("invalid value for state (must be 0..%d)", ERROR)}
			} else {
				if node, present := model.Nodes[nodeName]; present {
					node.Status = state
					outChannel <- OutChannelObject{Description: fmt.Sprintf("Set node %s to state %d", nodeName, state)}
				} else {
					outChannel <- OutChannelObject{Description: fmt.Sprintf("Node '%s' not defined", nodeName)}
				}
			}
		}
	default:
		outChannel <- OutChannelObject{Description: fmt.Sprintf("unrecognised command: %s", input.Cmd)}
	}
	return quitRequested
}

// ---------------------------------------

func _processAncestors(modelMap map[string]*Node, yamlNode YamlNode) ([]*Node, error) {
	var nodes []*Node
	for _, depends := range yamlNode.Depends {
		if yamlNode.Name == depends {
			return nil, fmt.Errorf("node '%s' references itself", yamlNode.Name)
		}
		if node, present := modelMap[depends]; present {
			nodes = append(nodes, node)
		} else {
			return nil, fmt.Errorf("node '%s' references unknown dependency '%s'", yamlNode.Name, depends)
		}
	}
	return nodes, nil
}
