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
}

// Print returns the statistics as a string
func (stats Statistics) Print() string {
	str := fmt.Sprintf("\nFinished in %s\n", stats.FinishTime.Sub(stats.StartTime).Round(time.Second))

	// get length of longest projectname
	maxLength := 0
	for _, node := range stats.NodeStats {
		maxLength = max(len(node.Name), maxLength)
	}
	maxLength += 2
	filler := strings.Repeat(".", maxLength)

	for _, node := range stats.NodeStats {
		myFiller := filler[:maxLength-len(node.Name)]
		str += fmt.Sprintf("%s%s %v\n", node.Name, myFiller, node.BuildTime)
	}

	return str
}
