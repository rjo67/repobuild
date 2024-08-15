package repobuild

import (
	"fmt"
	"testing"
)

func Test_createModel(t *testing.T) {
	tests := []struct {
		name      string
		yamlModel YamlModel
		want      string
		wantErr   bool
	}{
		{"ok", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"B"}}, {Name: "B"}}}, "map[A:(A/[(B/[]/[])]/[]) B:(B/[]/[])]", false},
		{"unknown child", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"B"}}}}, "", true},
		{"node references itself", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"A"}}}}, "", true},
		{"same node declared twice", YamlModel{Nodes: []YamlNode{{Name: "A"}, {Name: "A"}}}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewModel(tt.yamlModel)
			if (err != nil) != tt.wantErr {
				t.Errorf("createModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			gotStr := fmt.Sprintf("%v", got.Nodes)
			if gotStr != tt.want {
				t.Errorf("createModel() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func Test_checkMemory(t *testing.T) {
	inputyaml := `nodes:
  - name: A
    depends: [B, C]
  - name: B
    depends: [C]
  - name: C
    depends: []`

	yModel, err := LoadYamlModel([]byte(inputyaml))
	if err != nil {
		t.Errorf("loadYamlModel() error = %v", err)
		return
	}
	model, err := NewModel(yModel)
	if err != nil {
		t.Errorf("createModel() error = %v", err)
		return
	}
	var dependencyInNodeA *Node
	var dependencyInNodeB *Node
	for _, node := range model.Nodes {
		if node.Name == "A" {
			for _, depends := range node.Ancestors {
				if depends.Name == "C" {
					dependencyInNodeA = depends
					break
				}
			}
		}
		if node.Name == "B" {
			for _, depends := range node.Ancestors {
				if depends.Name == "C" {
					dependencyInNodeB = depends
					break
				}
			}
		}
	}
	if dependencyInNodeA != dependencyInNodeB {
		t.Error("the 'C' dependency in nodes A and B was not the same object")
		return
	}
}

func TestModel_detectCycle(t *testing.T) {
	tests := []struct {
		name      string
		yamlModel YamlModel
		wantErr   bool
	}{
		{"cycle 2 nodes", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"A"}}}}, true},
		{"cycle 3 nodes", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"A"}}}}, true},
		{"cycle B-C", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"B"}}}}, true},
		{"cycle B-C 2ndversion", YamlModel{Nodes: []YamlNode{{Name: "A", Depends: []string{}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"B"}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewModel(tt.yamlModel)
			if err != nil {
				t.Errorf("could not create model, error = %v", err)
			}

			err = model.DetectCycle()
			if (err != nil) != tt.wantErr {
				t.Errorf("Model.detectCycle() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.Logf("got error=%v", err)
		})
	}
}
