package main

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Note: struct fields must be public in order for unmarshal to correctly populate the data.
type YamlModel struct {
	Name string
	Data []YamlProject
}
type YamlProject struct {
	Name    string
	Depends []string // these must be processed before this project can be built, i.e. ancestors
}

type Node struct {
	Name      string
	Ancestors []*Node
	Children  []*Node
}

func (n Node) String() string {
	return fmt.Sprintf("(%s/%s/%s)", n.Name, n.Ancestors, n.Children)
}

func stringAddr(obj any) string {
	return fmt.Sprintf("%p_%s_", &obj, obj)
}

func loadYamlModel(data []byte) (YamlModel, error) {
	model := YamlModel{}
	err := yaml.Unmarshal(data, &model)
	return model, err
}

func createModel(yamlModel YamlModel) ([]Node, error) {
	modelMap := make(map[string]*Node)
	var err error
	// first create all Node objects in map
	for _, yamlProject := range yamlModel.Data {
		projectName := yamlProject.Name
		if _, present := modelMap[projectName]; present {
			return nil, fmt.Errorf("project '%s' defined multiple times", projectName)
		}
		node := Node{Name: projectName}
		modelMap[projectName] = &node
	}
	// process dependencies
	for _, yamlProject := range yamlModel.Data {
		projectName := yamlProject.Name
		if ancestors, err := _processAncestors(modelMap, yamlProject); err != nil {
			return nil, err
		} else {
			node := modelMap[projectName]
			node.Ancestors = ancestors
		}
	}

	model := make([]Node, 0, len(modelMap))
	for _, value := range modelMap {
		model = append(model, *value)
	}
	return model, err
}

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
