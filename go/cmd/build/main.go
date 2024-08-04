package main

import (
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/rjo67/repobuild"
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
		fmt.Println("must specify name of node definition file")
		os.Exit(1)
	}

	var model repobuild.Model
	var err error
	if model, err = process(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	out := make(chan repobuild.OutChannelObject)
	in := make(chan repobuild.InChannelObject)
	var wg = sync.WaitGroup{}
	wg.Add(2)
	go repobuild.ModelProcessor(model, in, out, &wg)
	go repobuild.TestDriver(out, in, &wg)

	wg.Wait()
}

// process reads in the input file and returns a Model
func process(args Args) (repobuild.Model, error) {
	inputData, err := os.ReadFile(args.Filename)
	if err != nil {
		return repobuild.Model{}, err
	}
	yamlModel, err := repobuild.LoadYamlModel(inputData)
	if err != nil {
		return repobuild.Model{}, err
	}
	model, err := repobuild.CreateModel(yamlModel)
	if err != nil {
		return repobuild.Model{}, err
	}

	return model, nil
}
