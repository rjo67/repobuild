package main

import (
	"fmt"
	"testing"
)

func Test_loadModel(t *testing.T) {
	tests := []struct {
		name      string
		inputyaml string
		want      string
		wantErr   bool
	}{
		{"one-project",
			`data:
  - name: A
    depends: [B, C]`,
			"{ [{A [B C]}]}", false},
		{"two-projects",
			`data:
  - name: A
    depends: [B, C]
  - name: B
    depends: [C]`,
			"{ [{A [B C]} {B [C]}]}", false},
		{"three-projects",
			`data:
  - name: A
    depends: [B, C]
  - name: B
    depends: []
  - name: C
    depends: [D]`,
			"{ [{A [B C]} {B []} {C [D]}]}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadYamlModel([]byte(tt.inputyaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("loadModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr := fmt.Sprintf("%v", got)
			if gotStr != tt.want {
				t.Errorf("loadModel() = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func Test_createModel(t *testing.T) {
	tests := []struct {
		name      string
		yamlModel YamlModel
		want      string
		wantErr   bool
	}{
		{"ok", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}, {Name: "B"}}}, "[(A/[(B/[]/[])]/[]) (B/[]/[])]", false},
		{"unknown child", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"B"}}}}, "", true},
		{"project references itself", YamlModel{Data: []YamlProject{{Name: "A", Depends: []string{"A"}}}}, "", true},
		{"same project declared twice", YamlModel{Data: []YamlProject{{Name: "A"}, {Name: "A"}}}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createModel(tt.yamlModel)
			if (err != nil) != tt.wantErr {
				t.Errorf("createModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			gotStr := fmt.Sprintf("%v", got)
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

	yModel, err := loadYamlModel([]byte(inputyaml))
	if err != nil {
		t.Errorf("loadYamlModel() error = %v", err)
		return
	}
	model, err := createModel(yModel)
	if err != nil {
		t.Errorf("createModel() error = %v", err)
		return
	}
	var dependencyInProjectA *Node
	var dependencyInProjectB *Node
	for _, node := range model {
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
