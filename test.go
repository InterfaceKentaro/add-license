package main

import (
	"log"
	"os"
)

func main() {
	var (
		file *os.File
		err error
		path string = "notexist.txt"
	)
	file, err = os.OpenFile(path, os.O_RDWR, 644)
	if (err != nil) {
		log.Fatalf("Failed to open a file. %v", err)
	}
	defer file.Close()
}
