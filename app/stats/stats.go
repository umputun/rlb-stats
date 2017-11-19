package stats

import "time"

// File contain single file hit count
type File struct {
	name string
	hits int
}

// Node contain single node hit count
type Node struct {
	name string
	hits int
}

// FilePerPeriod returns download stats for given filename at given period of time
func FilePerPeriod(filename string, periodStart, periodEnd time.Time) (file File) {
	return
}

// ByNodePerPeriod returns download stats for given period of time divided by nodes
func ByNodePerPeriod(periodStart, periodEnd time.Time) (nodes []Node) {
	return
}

// ByFilePerPeriod returns download stats for given period of time divided by files
func ByFilePerPeriod(periodStart, periodEnd time.Time) (files []File) {
	return
}
