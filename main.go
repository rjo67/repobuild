package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
)

type Args struct {
	Filename string
}

// InChannelObject is a general purpose struct which gets passed on the input channel to the modelProcessor
type InChannelObject struct {
	cmd  string // name of command
	data string // data for command
}

// OutChannelObject is a general purpose struct which gets passed on the output channel from the modelProcessor
type OutChannelObject struct {
	description string
	nodeNames   []string
	nodeDesc    [2][]string
}

func main() {
	args := Args{}

	flag.StringVar(&args.Filename, "f", "", "node definition file (yaml)")
	flag.Parse()

	// check input parameters
	if len(args.Filename) <= 0 {
		fmt.Println("must specify name of node definition file")
		os.Exit(1)
	}

	var model Model
	var err error
	if model, err = process(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out := make(chan OutChannelObject)
	in := make(chan InChannelObject)
	var wg = sync.WaitGroup{}
	wg.Add(2)
	go modelProcessor(model, in, out, &wg)
	go testDriver(out, in, &wg)

	wg.Wait()
}

// process reads in the input file and returns a Model
func process(args Args) (Model, error) {
	inputData, err := os.ReadFile(args.Filename)
	if err != nil {
		return Model{}, err
	}
	yamlModel, err := loadYamlModel(inputData)
	if err != nil {
		return Model{}, err
	}
	model, err := createModel(yamlModel)
	if err != nil {
		return Model{}, err
	}

	return model, nil
}
