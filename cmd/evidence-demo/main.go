package main

import (
	"fmt"
	"github.com/RajChodisetti/incident-trial-lab/internal/presentation"
	"os"
)

func main() {
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "evidence-demo accepts no arguments")
		os.Exit(1)
	}
	root, err := os.Getwd()
	if err == nil {
		err = presentation.Generate(root)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Evidence demo generation failed; recorded inputs require review.")
		os.Exit(1)
	}
	fmt.Println("Validated recorded evidence. Local pages: .build/demo/index.html, comparison.html, provenance.html")
}
