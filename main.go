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

	// check input parameters
	if len(filename) <= 0 {
		fmt.Println("must specify filename")
		os.Exit(1)
	}

	// run program

	if err := run(filename); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(filename string) error {
	inputData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	yamlModel, err := loadYamlModel(inputData)
	if err != nil {
		return err
	}
	model, err := createModel(yamlModel)
	if err != nil {
		return err
	}

	fmt.Printf("%s", model.status())

	return nil
}

func stringAddr(obj any) string {
	return fmt.Sprintf("%p_%s_", &obj, obj)
}
