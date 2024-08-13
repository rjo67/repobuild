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
		{"ok", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}, {Name: "B"}}}, "map[A:(A/[(B/[]/[])]/[]) B:(B/[]/[])]", false},
		{"unknown child", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}}}, "", true},
		{"project references itself", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"A"}}}}, "", true},
		{"same project declared twice", YamlModel{Data: []YamlProject{{Name: "A"}, {Name: "A"}}}, "", true},
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
	inputyaml := `data:
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
	var dependencyInProjectA *Node
	var dependencyInProjectB *Node
	for _, node := range model.Nodes {
		if node.Name == "A" {
			for _, depends := range node.Ancestors {
				if depends.Name == "C" {
					dependencyInProjectA = depends
					break
				}
			}
		}
		if node.Name == "B" {
			for _, depends := range node.Ancestors {
				if depends.Name == "C" {
					dependencyInProjectB = depends
					break
				}
			}
		}
	}
	if dependencyInProjectA != dependencyInProjectB {
		t.Error("the 'C' dependency in projects A and B was not the same object")
		return
	}
}

func TestModel_detectCycle(t *testing.T) {
	tests := []struct {
		name      string
		yamlModel YamlModel
		wantErr   bool
	}{
		{"cycle 2 nodes", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"A"}}}}, true},
		{"cycle 3 nodes", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"A"}}}}, true},
		{"cycle B-C", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"B"}}}}, true},
		{"cycle B-C 2ndversion", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{}}, {Name: "B", Depends: []string{"C"}}, {Name: "C", Depends: []string{"B"}}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model, err := NewModel(tt.yamlModel)
			if err != nil {
				t.Errorf("could not create model, error = %v", err)
			}

			err = model.detectCycle()
			if (err != nil) != tt.wantErr {
				t.Errorf("Model.detectCycle() error = %v, wantErr %v", err, tt.wantErr)
			}
			t.Logf("got error=%v", err)
		})
	}
}
