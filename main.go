package main

import (
	"fmt"

	"os"

	"github.com/suman9054/supersand/healper"
	"github.com/suman9054/supersand/process/container"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {

		if err := container.Runcontainer(os.Args[2]); err != nil {
			// Write directly to stderr so the Parent PTY can catch it
			fmt.Println("something is went wrong", err)
		}
		return
	}
	healper.PrintBanner()

}
