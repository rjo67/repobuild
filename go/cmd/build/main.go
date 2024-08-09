package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rjo67/repobuild"
)

type Args struct {
	Filename string
	StartCli bool // start the command line process
	Ignored  string
}

func main() {
	args := Args{}

	flag.StringVar(&args.Filename, "f", "", "node definition file (yaml)")
	flag.BoolVar(&args.StartCli, "c", false, "start command line interface")
	flag.StringVar(&args.Ignored, "i", "", "list of projects to ignore (comma-separated)")
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

	if args.Ignored != "" {
		nbr, err := model.SetIgnored(args.Ignored)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		} else {
			projectStr := "project"
			if nbr != 1 {
				projectStr += "s"
			}
			fmt.Printf("Set %d %s to state 'ignored'\n", nbr, projectStr)
		}
	}

	cliCommunication := repobuild.CliCommunication{ToCli: make(chan repobuild.OutChannelObject), FromCli: make(chan repobuild.InChannelObject)}
	var wg = sync.WaitGroup{}

	stats := repobuild.Statistics{StartTime: time.Now()}
	wg.Add(1)
	go repobuild.ModelProcessor(model, &cliCommunication, &stats, &wg)
	if args.StartCli {
		wg.Add(1)
		cliCommunication.StopChan = make(chan int)
		go repobuild.ModelCli(&cliCommunication, &wg)
	}

	wg.Wait()
	stats.FinishTime = time.Now()
	fmt.Printf("%s\n", stats.Print())
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
