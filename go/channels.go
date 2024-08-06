package repobuild

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

// CliCommunication stores the communication channels between the CLI and the model-processor
type CliCommunication struct {
	FromCli  chan InChannelObject
	ToCli    chan OutChannelObject
	StopChan chan int // used by the model-processor to signal 'stop' to the CLI
}
