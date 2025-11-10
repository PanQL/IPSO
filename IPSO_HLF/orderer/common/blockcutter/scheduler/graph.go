package scheduler

type Node struct {
	index int32
	txids []string // TODO: optimize by using bitset
	weight int
	// weight = len(txids)
	readSet  []uint64
	writeSet []uint64
	deltaSet []uint64
}
