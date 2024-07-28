package main

import (
	"fmt"
)

// Status values
const (
	WAITING  = iota
	RUNNING  = iota
	FINISHED = iota
	ERROR    = iota
)

type Model struct {
	Nodes map[string]*Node
}

/*
nodesWithStatus returns the nodes with the required status
*/
func (m Model) nodesWithStatus(requiredStatus int) []*Node {
	var nodes []*Node
	for _, node := range m.Nodes {
		if node.Status == requiredStatus {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

/*
status returns an overview of all nodes and their stati
*/
func (m Model) status() string {
	waiting := len(m.nodesWithStatus(WAITING))
	running := len(m.nodesWithStatus(RUNNING))
	finished := len(m.nodesWithStatus(FINISHED))
	errorStatus := len(m.nodesWithStatus(ERROR))
	var errStr = ""
	if errorStatus != 0 {
		errStr = "in error, "
	}
	total := waiting + running + finished + errorStatus
	return fmt.Sprintf("%d waiting, %d running, %d finished, "+errStr+"%d in total", waiting, running, finished, total)
}

type Node struct {
	Name      string
	Status    int
	Ancestors []*Node
	Children  []*Node
}

func (n Node) String() string {
	return fmt.Sprintf("(%s/%s/%s)", n.Name, n.Ancestors, n.Children)
}

func createModel(yamlModel YamlModel) (Model, error) {
	modelMap := make(map[string]*Node)
	var err error
	// first create all Node objects in map
	for _, yamlProject := range yamlModel.Data {
		projectName := yamlProject.Name
		if _, present := modelMap[projectName]; present {
			return Model{}, fmt.Errorf("project '%s' defined multiple times", projectName)
		}
		node := Node{Name: projectName, Status: WAITING}
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

	/*
		nodes := make([]Node, 0, len(modelMap))
		for _, value := range modelMap {
			nodes = append(nodes, *value)
		}
	*/
	return Model{Nodes: modelMap}, err
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
