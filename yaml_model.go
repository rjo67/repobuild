package main

import (
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

func loadYamlModel(data []byte) (YamlModel, error) {
	model := YamlModel{}
	err := yaml.Unmarshal(data, &model)
	return model, err
}
