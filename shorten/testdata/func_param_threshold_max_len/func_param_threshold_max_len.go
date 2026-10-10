package fixtures

// Short line, 3 params: threshold fires, line-length does not.

func shortLineThreeParams(a int, b string, c bool) error {
	return nil
}

// Long line, 2 params: line-length fires, threshold does not.

func longLineTwoParams(aReallyLongParamName string, anotherLongParamName string) error {
	return nil
}

// Long line, 3 params: both triggers fire; single clean expansion.

func longLineThreeParams(aReallyLongParamName string, anotherLongParamName string, c bool) error {
	return nil
}
