package repobuild

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Note: struct fields must be public in order for unmarshal to correctly populate the data.
type YamlModel struct {
	Name string
	Nodes []YamlNode
}
type YamlNode struct {
	Name    string
	Ignore  bool     // will not try to 'build' the node
	Script  string   // the script to run in order to build the node
	Depends []string // these must be processed before this node can be built, i.e. ancestors
}

func (y YamlNode) String() string {
	script := ""
	if y.Script != "" {
		script = fmt.Sprintf(" '%s'", y.Script)
	}
	ignore := ""
	if y.Ignore {
		ignore = " (ignored)"
	}
	return fmt.Sprintf("{%s%s %v%s}", y.Name, script, y.Depends, ignore)
}

// LoadYamlModel parses the yaml input file
func LoadYamlModel(data []byte) (YamlModel, error) {
	model := YamlModel{}
	//err := yaml.Unmarshal(data, &model)

	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true) // Disallow unknown fields
	err := decoder.Decode(&model)
	if err != nil {
		err = fmt.Errorf("parse error: %v", err)
	}
	return model, err
}
