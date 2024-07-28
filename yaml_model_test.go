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
