package fixtures

func chainSegmentArgs() {
	m.EXPECT().
		Get(1).
		Return(&Result{Stdout: "a really long standard output value here"}, &Result{Stdout: "another really long standard output value"}, nil)
}

func chainMixedSegments() {
	m.EXPECT().
		Get(1, "short").
		Process("a really really really long argument that pushes this line over the limit", "and another fairly long one").
		Done("x", "y")
}
