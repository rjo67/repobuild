package repobuild

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
	Script  string   // the script to run in order to build the project
	Depends []string // these must be processed before this project can be built, i.e. ancestors
}

func (y YamlProject) String() string {
	script := ""
	if y.Script != "" {
		script = fmt.Sprintf(" '%s'", y.Script)
	}
	return fmt.Sprintf("{%s%s %v}", y.Name, script, y.Depends)
}

// LoadYamlModel parses the yaml input file
func LoadYamlModel(data []byte) (YamlModel, error) {
	model := YamlModel{}
	err := yaml.Unmarshal(data, &model)
	return model, err
}
