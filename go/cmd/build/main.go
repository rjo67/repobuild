package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rjo67/repobuild"
	"github.com/rjo67/repobuild/cli"
)

type Args struct {
	Filename        string
	InteractiveMode bool // will wait for commands from cli
	Ignored         string
}

func main() {
	args := Args{}

	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `Usage:
  -f, --file file
        node definition file (yaml)
  -i, --ignore nodes
        comma-separated list of nodes to ignore
  --interactive
        Start in interactive mode.
        Waits for commands via the command line interface.`)
	}
	flag.StringVar(&args.Filename, "f", "", "node definition `file` (yaml)")
	flag.StringVar(&args.Filename, "file", "", "node definition `file` (yaml)")
	flag.BoolVar(&args.InteractiveMode, "interactive", false, "interactive mode")
	flag.StringVar(&args.Ignored, "i", "", "comma-separated list of `nodes` to ignore")
	flag.StringVar(&args.Ignored, "ignore", "", "comma-separated list of `nodes` to ignore")
	flag.Parse()

	// check input parameters
	if len(args.Filename) <= 0 {
		fmt.Println("must specify name of node definition file")
		os.Exit(1)
	}

	var model *repobuild.Model
	var err error
	model, err = process(args)
	if err == nil {
		if args.Ignored != "" {
			var nbr int
			nbr, err = model.SetIgnored(args.Ignored)
			if err == nil {
				nodeStr := "node"
				if nbr != 1 {
					nodeStr += "s"
				}
				fmt.Printf("Set %d %s to state 'ignored'\n", nbr, nodeStr)
			}
		}
		if err == nil {
			err = model.DetectCycle()
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cliCommunication := repobuild.CliCommunication{ToCli: make(chan repobuild.OutChannelObject), FromCli: make(chan repobuild.InChannelObject)}
	var wg = sync.WaitGroup{}

	stats := repobuild.Statistics{StartTime: time.Now()}

	//
	// The ModelProcessor and the CLI are always started.
	// Normally the ModelProcessor will automatically start processing.
	// In 'interactive mode', the ModelProcessor will wait for commands from the CLI.
	//
	wg.Add(1)
	go model.ModelProcessor(args.InteractiveMode, &cliCommunication, &stats, &wg)
	wg.Add(1)
	cliCommunication.StopChan = make(chan int)
	portstr := "3333"
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Fatal(err)
	}
	ip, err := net.LookupIP("localhost")
	if err != nil {
		log.Fatal(err)
	}
	go startCliServer(&cliCommunication, &wg, ip[0], port)

	wg.Wait()
	stats.FinishTime = time.Now()
	fmt.Printf("%s\n", stats.Print())
}

func startCliServer(cliCommunication *repobuild.CliCommunication, wg *sync.WaitGroup, ip net.IP, port int) {
	cliServer := cli.New(ip, port)
	cliServer.Run(cliCommunication, wg)
}

// process reads in the input file and returns a Model
func process(args Args) (*repobuild.Model, error) {
	inputData, err := os.ReadFile(args.Filename)
	if err != nil {
		return nil, err
	}
	yamlModel, err := repobuild.LoadYamlModel(inputData)
	if err != nil {
		return nil, err
	}
	model, err := repobuild.NewModel(yamlModel)
	if err != nil {
		return nil, err
	}

	return model, nil
}
