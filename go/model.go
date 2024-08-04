package repobuild

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Status values
const (
	WAITING  = iota
	RUNNING  = iota
	FINISHED = iota
	ERROR    = iota
)

func statusToString(status int) string {
	switch status {
	case WAITING:
		return "waiting"
	case RUNNING:
		return "running"
	case FINISHED:
		return "finished"
	case ERROR:
		return "in error"
	}
	return fmt.Sprintf("status not recognised: %d", status)
}

type Model struct {
	Nodes map[string]*Node
}

// nodesWithStatus returns the nodes with the required status
func (m Model) nodesWithStatus(requiredStatus int) []*Node {
	var nodes []*Node
	for _, node := range m.Nodes {
		if node.Status == requiredStatus {
			nodes = append(nodes, node)
		}
	}
	return nodes
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
	waitingNodes := m.nodesWithStatus(WAITING)
	waiting := len(waitingNodes)
	waitingNodeNames := getNodeNames(waitingNodes)
	runningNodes := m.nodesWithStatus(RUNNING)
	running := len(runningNodes)
	runningNodeNames := getNodeNames(runningNodes)
	finishedNodes := m.nodesWithStatus(FINISHED)
	finished := len(finishedNodes)
	finishedNodeNames := getNodeNames(finishedNodes)
	errorStatusNodes := m.nodesWithStatus(ERROR)
	errorStatus := len(errorStatusNodes)
	var errStr = ""
	if errorStatus != 0 {
		if verbose {
			errorStatusNodeNames := getNodeNames(errorStatusNodes)
			errStr = fmt.Sprintf("%d in error: %s\n", errorStatus, errorStatusNodeNames)
		} else {
			errStr = fmt.Sprintf("%d in error, ", errorStatus)
		}
	}
	total := waiting + running + finished + errorStatus
	if verbose {
		return fmt.Sprintf("%d waiting: %s\n%d running: %s\n%d finished: %s\n"+errStr+"TOTAL: %d", waiting,
			waitingNodeNames, running, runningNodeNames, finished, finishedNodeNames, total)
	} else {
		return fmt.Sprintf("%d waiting, %d running, %d finished, "+errStr+"TOTAL: %d", waiting, running, finished, total)
	}
}

// findRunnableNodes returns a 2 dimensional array of nodes.
// The first dimension contains the nodes which can be started.
// The second dimension contains those nodes which cannot be started.
// A node is startable if all its ancestors have status=FINISHED.
// The stateDescription array describes why the nodes can or cannot be started.
func (m Model) findRunnableNodes() (nodes [2][]*Node, stateDescription [2][]string) {
	for _, node := range m.nodesWithStatus(WAITING) {
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
	for _, desc := range stateDescription {
		sort.Strings(desc)
	}
	for _, node := range nodes {
		sort.Slice(node, func(i, j int) bool { return node[i].Name < node[j].Name })
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

// ancestorsAreFinished returns whether all the ancestors of this node have state FINISHED.
// The second parameter describes the state of all ancestors.
func (n Node) ancestorsAreFinished() (allAncestorsFinished bool, desc string) {
	allAncestorsFinished = true
	var ancestorStatus [ERROR + 1][]string
	for _, node := range n.Ancestors {
		ancestorStatus[node.Status] = append(ancestorStatus[node.Status], node.Name)
		if node.Status != FINISHED {
			allAncestorsFinished = false
		}
	}
	for status := WAITING; status <= ERROR; status++ {
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
		node := Node{Name: projectName, Script: yamlProject.Script, Status: WAITING}
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

// ModelProcessor is a simple loop to receive commands via the in channel and send the results back on the out channel
func ModelProcessor(model Model, inChannel chan InChannelObject, out chan OutChannelObject, wg *sync.WaitGroup) {
	for input := range inChannel {
		switch input.cmd {
		case "status":
			out <- OutChannelObject{description: model.status(true)}
		case "findRunnableNodes":
			nodes, nodesDesc := model.findRunnableNodes()
			var nodeStr []string
			for _, node := range nodes[0] {
				nodeStr = append(nodeStr, node.Name)
			}
			out <- OutChannelObject{nodeNames: nodeStr, nodeDesc: nodesDesc}
		case "set":
			split := strings.Split(strings.TrimSpace(input.data), " ")
			if len(split) != 2 { // should be 3: "<node> <state>"
				out <- OutChannelObject{description: fmt.Sprintf("invalid format, must be '<node> <state>', got: %s", input.data)}
			} else {
				nodeName := split[0]
				state, err := strconv.Atoi(split[1])
				if err != nil || state < 0 || state > ERROR {
					out <- OutChannelObject{description: fmt.Sprintf("invalid value for state (must be 0..%d)", ERROR)}
				} else {
					if node, present := model.Nodes[nodeName]; present {
						node.Status = state
						out <- OutChannelObject{description: fmt.Sprintf("Set node %s to state %d", nodeName, state)}
					} else {
						out <- OutChannelObject{description: fmt.Sprintf("Node '%s' not defined", nodeName)}
					}
				}
			}
		default:
			out <- OutChannelObject{description: fmt.Sprintf("unrecognised command: %s", input.cmd)}
		}
	}
	wg.Done()
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

// callScript executes the given scriptname and returns when the script finishes. Returns an error object (nil if RC==0)
func callScript(scriptname string) error {
	cmd := exec.Command("cmd.exe", "/C", scriptname)
	cmd = exec.Command(scriptname, "5")
	//cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	//	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return err
}
