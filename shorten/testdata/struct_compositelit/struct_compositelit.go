package struct_compositelit

type Foo struct {
	field1 string
	field2 string
	field3 []string
	field4 string
	field5 string
}

const floccinaucinihilipilification = "floccinaucinihilipilification"
const supercalifragilisticexpialidocious = "Supercalifragilisticexpialidocious"

var _ = &Foo{
	field1: "Pneumonoultramicroscopicsilicovolcanoconiosis", field2: helloWorld(supercalifragilisticexpialidocious), field3: []string{"list1"}, field4: floccinaucinihilipilification, field5: "foo",
}

var _ = &Foo{
	field1: "a", field2: helloWorld("a"), field3: []string{"list1"}, field4: "c", field5: "foo",
}

func helloWorld(_ string) string {
	return "hello world"
}

func _() {
	_ := &Foo{
		field1: "Pneumonoultramicroscopicsilicovolcanoconiosis", field2: helloWorld(supercalifragilisticexpialidocious), field3: []string{"list1"}, field4: floccinaucinihilipilification, field5: "foo",
	}
}
