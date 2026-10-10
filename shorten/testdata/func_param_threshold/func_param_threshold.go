package fixtures

// Functions below threshold: unchanged.

func zeroParams() error {
	return nil
}

func oneParam(a int) error {
	return nil
}

func twoParams(a int, b string) error {
	return nil
}

// Functions at or above threshold: expand.

func threeParams(a int, b string, c bool) error {
	return nil
}

func fourParams(a int, b string, c bool, d float64) error {
	return nil
}

// Methods: receiver is not counted toward the threshold.

type Receiver struct{}

func (r *Receiver) methodTwoParams(a int, b string) error {
	return nil
}

func (r *Receiver) methodThreeParams(a int, b string, c bool) error {
	return nil
}

// Value receiver on a named basic type: receiver still not counted.

type MyString string

func (s MyString) basicTypeTwoParams(a int, b string) error {
	return nil
}

func (s MyString) basicTypeThreeParams(a int, b string, c bool) error {
	return nil
}

// Named params with shared types: count names, not fields.
// a, b int, c string = 3 logical params.

func sharedTypes(a, b int, c string) error {
	return nil
}

// Unnamed params.

func unnamed(int, string, bool) error {
	return nil
}

// Variadic: a + b = 2 logical params, unchanged.

func variadic(a int, b ...string) error {
	return nil
}

// Already multi-line: idempotent, no change.

func alreadyMultiLine(
	a int,
	b string,
	c bool,
) error {
	return nil
}

// Return values are not counted toward the threshold.

func returnValuesNotCounted(a int) (int, string, bool) {
	return 0, "", false
}

// Interface methods.

type MyInterface interface {
	interfaceMethodTwo(a int, b string) error
	interfaceMethodThree(a int, b string, c bool) error
}

// Function type declarations.

type MyFunc func(a int, b string, c bool) error

// Function literals.

func funcLiterals() {
	_ = func(a int, b string) error {
		return nil
	}
	_ = func(a int, b string, c bool) error {
		return nil
	}
}
