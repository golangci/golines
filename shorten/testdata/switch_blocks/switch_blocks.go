package fixtures

import "fmt"

func _(a int) error {
	c := make(chan int)

	for {
		select {
		case <-c:
			switch a {
			case 1:
				return fmt.Errorf("This is a really long line that can be broken up twice %s %s", fmt.Sprintf("This is a really long sub-line that should be broken up more because %s %s", "xxxx", "yyyy"), fmt.Sprintf("A short one %d", 3))
			case 2:
			}
		}

		break
	}

	return nil
}

func _() {
	var pneumonoultramicroscopicsilicovolcanoconiosis string
	var floccinaucinihilipilification string

	switch myfunction(pneumonoultramicroscopicsilicovolcanoconiosis, floccinaucinihilipilification) {
	case "a":
		fmt.Println("a")
	}
}

func _() {
	var pneumonoultramicroscopicsilicovolcanoconiosis string
	var floccinaucinihilipilification string

	switch a := "taumatawhakatangihangakoauauotamateaturipukakapikimaungahoronukupokaiwhenuakitanatahu"; myfunction(pneumonoultramicroscopicsilicovolcanoconiosis, floccinaucinihilipilification) {
	case "a":
		fmt.Println(a)
	}
}

func myfunction(a, b string) string {
	return ""
}
