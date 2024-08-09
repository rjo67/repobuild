package repobuild

import (
	"fmt"
	"math/rand/v2"
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

type Model struct {
	Nodes map[string]*Node
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

// SetIgnored sets the 'ignored' flag on the projects in the eparated list 'ignored'
// Returns the number of projects processed
func (m Model) SetIgnored(ignored string) (int, error) {
	projectNames := strings.Split(ignored, ",")
	for _, name := range projectNames {
		if node, present := m.Nodes[name]; present {
			node.Status = IGNORED
		} else {
			return -1, fmt.Errorf("unknown project '%s' in ignore list", name)
		}
	}
	return len(projectNames), nil
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
	if verbose {
		return fmt.Sprintf("%d waiting: %s\n%d running: %s\n%d finished: %s\n%s%sTOTAL: %d", nbrNodes[WAITING],
			nodeNames[WAITING], nbrNodes[RUNNING], nodeNames[RUNNING], nbrNodes[FINISHED], nodeNames[FINISHED], errStr, ignoredStr, totalNbrNodes)
	} else {
		return fmt.Sprintf("%d waiting, %d running, %d finished, %s%sTOTAL: %d", nbrNodes[WAITING], nbrNodes[RUNNING], nbrNodes[FINISHED], errStr, ignoredStr, totalNbrNodes)
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

// detectCycle inspects the model to detect a possible cycle.
func (m Model) detectCycle() error {
	nameStack := Stack[string]()
	for _, node := range m.Nodes {
		alreadyProcessed := make(map[string]bool)
		alreadyProcessed[node.Name] = true
		nameStack.Push(node.Name)
		err := _detectCycle(node, &alreadyProcessed, nameStack)
		if err != nil {
			return err
		}
		nameStack.Pop()
	}
	return nil
}
func _detectCycle(node *Node, alreadyProcessed *map[string]bool, nameStack stack[string]) error {
	for _, ancestor := range node.Ancestors {
		if _, present := (*alreadyProcessed)[ancestor.Name]; present {
			nameStack.Push(ancestor.Name) // to improve error messagej
			return fmt.Errorf("cycle detected: %v", nameStack.Elements())
		}
		(*alreadyProcessed)[ancestor.Name] = true
		nameStack.Push(ancestor.Name)
		err := _detectCycle(ancestor, alreadyProcessed, nameStack)
		if err != nil {
			return err
		}
		nameStack.Pop()
	}
	return nil
}

type Node struct {
	Name      string
	Script    string
	Status    int
	Ancestors []*Node
	Children  []*Node
}

type Statistics struct {
	StartTime time.Time
	NodeStats []NodeStatistics
}
type NodeStatistics struct {
	Name      string
	BuildTime time.Duration
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

// CreateModel creates the internal model from the yaml model
func CreateModel(yamlModel YamlModel) (Model, error) {
	modelMap := make(map[string]*Node)
	var err error
	// first create all Node objects in map
	for _, yamlProject := range yamlModel.Data {
		projectName := yamlProject.Name
		if _, present := modelMap[projectName]; present {
			return Model{}, fmt.Errorf("project '%s' defined multiple times", projectName)
		}
		nodeStatus := WAITING
		if yamlProject.Ignore {
			nodeStatus = IGNORED
		}
		node := Node{Name: projectName, Script: yamlProject.Script, Status: nodeStatus}
		modelMap[projectName] = &node
	}
	// process dependencies
	for _, yamlProject := range yamlModel.Data {
		projectName := yamlProject.Name
		if ancestors, err := _processAncestors(modelMap, yamlProject); err != nil {
			return Model{}, err
		} else {
			node := modelMap[projectName]
			node.Ancestors = ancestors
		}
	}
	return Model{Nodes: modelMap}, err
}

// ModelProcessor is the main loop to process the model.
// Will loop 'forever' and process any nodes which can be run.
// Simple commands can be sent via the cliCommunoication 'in' Channel (results are sent back on the 'out' Channel)
// Statistics will be stored in stats.
func ModelProcessor(model Model, cliCommunication *CliCommunication, stats *Statistics, wg *sync.WaitGroup) {
	nodesChannel := make(chan string)
	nodeStatusChanged := false
	nodeStartTime := make(map[string]time.Time) // stores start times for nodes
	// preprocess 'ignored' nodes
	for _, ignoredNode := range model.nodesWithStatus()[IGNORED] {
		stats.NodeStats = append(stats.NodeStats, NodeStatistics{Name: ignoredNode.Name, BuildTime: time.Duration(0)})
	}
	for stopLoop := false; !stopLoop; {
		select {
		// get input from model cli
		case input := <-cliCommunication.FromCli:
			stopLoop = _processCommand(model, input, cliCommunication.ToCli)
		// get info from node subtasks
		case nodeInput := <-nodesChannel:
			fields := strings.Fields(nodeInput)
			if len(fields) == 2 {
				switch fields[0] {
				case "FINISHED":
					if node, present := model.Nodes[fields[1]]; present {
						if node.Status != RUNNING {
							fmt.Printf("Invalid status for node %s: %s\n", fields[1], statusToString(node.Status))
						} else {
							node.Status = FINISHED
							if startTime, present := nodeStartTime[node.Name]; present {
								stats.NodeStats = append(stats.NodeStats,
									NodeStatistics{Name: node.Name, BuildTime: time.Since(startTime).Round(time.Second)})
							}
							nodeStatusChanged = true
						}
					} else {
						fmt.Printf("Node %s not found\n", fields[1])
					}
				default:
					fmt.Printf("Unrecognised message on nodes channel: %s\n", nodeInput)
				}
			} else {
				fmt.Printf("Invalid message on nodes channel: %s\n", nodeInput)
			}
		// update state of model (have tasks finished? Start new tasks, etc)
		default:
			runnableNodes, _ := model.findRunnableNodes()
			for _, node := range runnableNodes[0] {
				nodeStartTime[node.Name] = time.Now()
				_startNode(node, nodesChannel)
				nodeStatusChanged = true
			}
		}
		if nodeStatusChanged {
			time.Sleep(SLEEP_WAIT_DURATION)
			fmt.Printf("\n\n%s\n", model.status(false))
			nodeStatusChanged = false
		}
		// TODO this exits the loop, if all finished, but the model-cli needs to be notified
		if len(model.Nodes) == model.nodesInEndStatus() {
			stopLoop = true
		}
	}
	// tell cli to stop (if present)
	if cliCommunication.StopChan != nil {
		cliCommunication.StopChan <- 1
		time.Sleep(300 * time.Millisecond)
	}
	wg.Done()
}

func _processCommand(model Model, input InChannelObject, outChannel chan<- OutChannelObject) (quitRequested bool) {
	quitRequested = false
	switch input.cmd {
	case "quit":
		quitRequested = true
	case "status":
		outChannel <- OutChannelObject{description: model.status(true)}
	case "findRunnableNodes":
		nodes, nodesDesc := model.findRunnableNodes()
		// just return the node names
		var nodeStr []string
		for _, node := range nodes[0] {
			nodeStr = append(nodeStr, node.Name)
		}
		outChannel <- OutChannelObject{nodeNames: nodeStr, nodeDesc: nodesDesc}
	case "set":
		split := strings.Split(strings.TrimSpace(input.data), " ")
		if len(split) != 2 { // should be 3: "<node> <state>"
			outChannel <- OutChannelObject{description: fmt.Sprintf("invalid format, must be '<node> <state>', got: %s", input.data)}
		} else {
			nodeName := split[0]
			state, err := strconv.Atoi(split[1])
			if err != nil || state < 0 || state > ERROR {
				outChannel <- OutChannelObject{description: fmt.Sprintf("invalid value for state (must be 0..%d)", ERROR)}
			} else {
				if node, present := model.Nodes[nodeName]; present {
					node.Status = state
					outChannel <- OutChannelObject{description: fmt.Sprintf("Set node %s to state %d", nodeName, state)}
				} else {
					outChannel <- OutChannelObject{description: fmt.Sprintf("Node '%s' not defined", nodeName)}
				}
			}
		}
	default:
		outChannel <- OutChannelObject{description: fmt.Sprintf("unrecognised command: %s", input.cmd)}
	}
	return quitRequested
}

func _startNode(node *Node, nodesChannel chan<- string) {
	fmt.Printf("starting node %s\n", (*node).Name)
	//time.Sleep(500 * time.Millisecond)
	go TestNode((*node).Name, nodesChannel)
	node.Status = RUNNING
}

// ---------------------------------------

func _processAncestors(modelMap map[string]*Node, yamlProject YamlProject) ([]*Node, error) {
	var nodes []*Node
	for _, depends := range yamlProject.Depends {
		if yamlProject.Name == depends {
			return nil, fmt.Errorf("project '%s' references itself", yamlProject.Name)
		}
		if node, present := modelMap[depends]; present {
			nodes = append(nodes, node)
		} else {
			return nil, fmt.Errorf("project '%s' references unknown dependency '%s'", yamlProject.Name, depends)
		}
	}
	return nodes, nil
}

func TestNode(name string, nodesChannel chan<- string) {
	sleep := rand.IntN(10) + 1
	fmt.Printf("Node %s starting (sleep=%d)\n", name, sleep)
	time.Sleep(time.Duration(sleep) * time.Second)
	fmt.Printf("Node %s finished\n", name)
	nodesChannel <- "FINISHED " + name
}
