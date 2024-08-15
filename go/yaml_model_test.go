package repobuild

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
		{"one-nodes",
			`nodes:
  - name: A
    depends: [B, C]`,
			"{ [{A [B C]}]}", false},
		{"two-nodes",
			`nodes:
  - name: A
    script: run.sh param1
    depends: [B, C]
  - name: B
    depends: [C]`,
			"{ [{A 'run.sh param1' [B C]} {B [C]}]}", false},
		{"three-nodes",
			`nodes:
  - name: A
    depends: [B, C]
  - name: B
    depends: []
  - name: C
    depends: [D]`,
			"{ [{A [B C]} {B []} {C [D]}]}", false},
		{"invalid field",
			`nodes:
  - name: A
    depends: [B, C],
	silly: true`,
			"{ []}", true},
		{"ignored node",
			`nodes:
  - name: A
    depends: [B, C]
  - name: B
    ignore: true`,
			"{ [{A [B C]} {B [] (ignored)}]}", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadYamlModel([]byte(tt.inputyaml))
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
