package nnue

// AccumulatorSnapshotForTest returns a value copy for independent test oracles.
// It is compiled only into tests and cannot mutate Context state.
func AccumulatorSnapshotForTest(c *Context) ([PerspectiveCount][HiddenSize]int64, error) {
	if err := c.ready(); err != nil {
		return [PerspectiveCount][HiddenSize]int64{}, err
	}
	return c.states[c.depth].accumulators, nil
}
