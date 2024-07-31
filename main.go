package main

import (
	"flag"
	"fmt"
	"os"
)

type Args struct {
	Filename string
}

func main() {
	args := Args{}

	flag.StringVar(&args.Filename, "f", "", "node definition file (yaml)")
	flag.Parse()

	// check input parameters
	if len(args.Filename) <= 0 {
		fmt.Println("must specify filename")
		os.Exit(1)
	}

	// run program

	if err := process(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func process(args Args) error {
	inputData, err := os.ReadFile(args.Filename)
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

	testDriver(model)

	return nil
}
