package repobuild

import (
	"fmt"
	"strings"
	"time"
)

type Statistics struct {
	StartTime  time.Time
	FinishTime time.Time
	NodeStats  []NodeStatistics
}
type NodeStatistics struct {
	Name      string
	BuildTime time.Duration
	Ignored   bool
}

// Print returns the statistics as a string
func (stats Statistics) Print() string {
	str := fmt.Sprintf("\nProcessed %d nodes in %s\n", len(stats.NodeStats), stats.FinishTime.Sub(stats.StartTime).Round(time.Second))

	// get length of longest nodename
	maxLength := 0
	for _, node := range stats.NodeStats {
		maxLength = max(len(node.Name), maxLength)
	}
	maxLength += 2
	filler := strings.Repeat(".", maxLength)

	for _, node := range stats.NodeStats {
		myFiller := filler[:maxLength-len(node.Name)]
		var desc string
		if node.Ignored {
			desc = "ignored"
		} else {
			desc = fmt.Sprintf("%v", node.BuildTime)
		}
		str += fmt.Sprintf("%s%s %s\n", node.Name, myFiller, desc)
	}

	return str
}
