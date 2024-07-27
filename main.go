package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {

	var filename string
	flag.StringVar(&filename, "f", "", "yaml file")
	flag.Parse()

	if len(filename) <= 0 {
		fmt.Println("must specify filename")
	} else {
		inputData, err := os.ReadFile(filename)
		if err != nil {
			fmt.Print(err)
		} else {
			yamlModel, err := loadYamlModel(inputData)
			if err != nil {
				fmt.Print(err)
			} else {
				fmt.Printf("YAML: %v\n", yamlModel)
				model, err := createModel(yamlModel)
				if err != nil {
					fmt.Print(err)
				} else {
					fmt.Printf("MODEL: %v\n", model)

				}
			}
		}
	}
}
