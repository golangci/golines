// Package main is used to generate code that creates dot files from golang ASTs.
package main

import (
	"log"
)

func main() {
	err := genNodeToGraphNode("shorten/internal/graph/graph_generated.go")
	if err != nil {
		log.Fatal(err)
	}
}
