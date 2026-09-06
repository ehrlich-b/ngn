package nnue

// AccumulatorSnapshotForTest returns a value copy for independent test oracles.
// It is compiled only into tests and cannot mutate Context state.
func AccumulatorSnapshotForTest(c *Context) ([PerspectiveCount][HiddenSize]int64, error) {
	if err := c.ready(); err != nil {
		return [PerspectiveCount][HiddenSize]int64{}, err
	}
	return c.states[c.depth].accumulators, nil
}

// Int32AccumulatorSnapshotForTest returns the narrow stored lanes without
// widening, so tests can verify both representation and numerical parity.
func Int32AccumulatorSnapshotForTest(c *Int32Context) ([PerspectiveCount][HiddenSize]int32, error) {
	if err := c.ready(); err != nil {
		return [PerspectiveCount][HiddenSize]int32{}, err
	}
	return c.states[c.depth].accumulators, nil
}
