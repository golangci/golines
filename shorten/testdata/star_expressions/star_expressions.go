package fixtures

func testStarExpressions() {
	values = append(values, *newValueFromInputs(firstArgument, secondArgument, thirdArgument, fourthArgument, fifthArgument, sixthArgument))
	result := *buildResult(firstArgument, secondArgument, thirdArgument, fourthArgument, fifthArgument, sixthArgument, seventhArgument)
	short := *shortCall(argument1)
}
