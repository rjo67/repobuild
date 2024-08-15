package repobuild

import (
	"fmt"
	"os"
	"testing"
)

func Test_loadModel(t *testing.T) {
	tests := []struct {
		name          string
		inputyamlfile string
		want          string
		wantErr       bool
	}{
		{"one-node", "testdata/one-node.yaml", "{ [{A [B C]}]}", false},
		{"two-nodes", "testdata/two-nodes.yaml", "{ [{A 'run.sh param1' [B C]} {B [C]}]}", false},
		{"three-nodes", "testdata/three-nodes.yaml", "{ [{A [B C]} {B []} {C [D]}]}", false},
		{"invalid field", "testdata/invalid-field.yaml", "{ []}", true},
		{"ignored node", "testdata/ignored-node.yaml", "{ [{A [B C]} {B [] (ignored)}]}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := loadFile(tt.inputyamlfile)
			if err != nil {
				t.Errorf("error = %v", err)
				return
			}
			got, err := LoadYamlModel(bytes)
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

func loadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}
