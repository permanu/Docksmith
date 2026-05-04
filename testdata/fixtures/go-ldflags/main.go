package main

import "fmt"

var Version = "dev"
var Commit = "none"

func main() {
	fmt.Printf("version=%s commit=%s\n", Version, Commit)
}
