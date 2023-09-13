package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	FlagFilepath string
)

func main() {
	// cli args
	flag.StringVar(&FlagFilepath, "file", "", "File path")
	flag.Parse()
	// read file
	file, err := os.ReadFile(FlagFilepath)
	if err != nil {
		panic(err)
	}
	// convert to bytes
	bin := []byte(file)
	length := len(bin)
	fmt.Print("{")
	for i, v := range bin {
		fmt.Print(v)
		if i < length {
			fmt.Print(",")
		}
	}
	fmt.Println("}")
}
