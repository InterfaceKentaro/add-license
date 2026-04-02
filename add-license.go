package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// 0. get a file list and start iterate over the list.
// 1. load a file and buffer contents.
// 2. add a license text at the top of the file.
// 3. append rest of the file contents.
// 4. flush a file and exit.

// possible errors
// - can't get a file list.
// - can't open files
// - permission error: can't write to the file

const (
	MaxBufferSize int64  = 10 * 1024 * 1024 // 10MB
	LicenseText   string = "/* \n * Copyright (c) 2026 Your Company. All rights reserved.\n * This file is subject to the terms and conditions defined in LICENSE.txt\n */\n\n"
)

func main() {
	var rootPath *string
	rootPath = flag.String("path", "", "The root directory of the Java project")
	flag.Parse()

	if *rootPath == "" {
		fmt.Println("Error: Please provide a project path using -path")
		os.Exit(1)
	}

	var (
		info os.FileInfo
		err error
	) 
	info, err = os.Stat(*rootPath)

	if os.IsNotExist(err) {
		fmt.Printf("Error: The path '%s' does not exist.\n", *rootPath)
		os.Exit(1)
	}
	
	var javaFiles []string
	javaFiles, err = findJavaFiles(*rootPath)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	var path string
	for _, path = range javaFiles {
		err = prependLicense(path, &info)
		if err != nil {
			fmt.Printf("Failed to process [%s]: %v\n", path, err)
			continue
		}
		fmt.Printf("Successfully updated: %s\n", path)
	}
}

func findJavaFiles(root string) ([]string, error) {
	var files []string
	var err error
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".java" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func prependLicense(filePath string, originalMode *os.FileInfo) error {
	var (
		file_ori *os.File
		err error
		info os.FileInfo
		// originalMode os.FileMode
	)
	// Get original info first to check permissions and mode
	info, err = os.Stat(filePath)
	if err != nil {
		return err
	}

	// Open for reading and writing
	file_ori, err = os.OpenFile(filePath, os.O_RDWR, (*originalMode).Mode())
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: ensure you have write access")
		} else {
			return fmt.Errorf("The file potentially not exist. %v", err)
		}
	}
	defer file_ori.Close()

	var originalContentReader io.Reader

	if info.Size() > MaxBufferSize {
		var tempFile *os.File
		tempFile, err = os.CreateTemp("", "java_buffer_")
		if err != nil {
			return err
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		_, err = io.Copy(tempFile, file_ori)
		if err != nil {
			return err
		}
		tempFile.Seek(0, 0)
		originalContentReader = tempFile
		
		// Large file writing
		return rewriteFileFromReader(filePath, LicenseText, originalContentReader, (*originalMode).Mode())
	} else {
		var content []byte
		content, err = io.ReadAll(file_ori)
		if err != nil {
			return err
		}
		// Small file writing
		return rewriteFile(filePath, LicenseText, content, (*originalMode).Mode())
	}
}

// Inherits permissions via originalMode
func rewriteFile(path string, header string, content []byte, mode os.FileMode) error {
	var newContent []byte
	newContent = append([]byte(header), content...)
	// Uses originalMode instead of hardcoded 0644
	return os.WriteFile(path, newContent, mode)
}

// Inherits permissions via originalMode
func rewriteFileFromReader(path string, header string, reader io.Reader, mode os.FileMode) error {
	var f *os.File
	var err error
	var writer *bufio.Writer

	// os.OpenFile with O_TRUNC allows us to apply the specific mode upon creation
	f, err = os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close()

	writer = bufio.NewWriter(f)
	writer.WriteString(header)
	_, err = io.Copy(writer, reader)
	if err != nil {
		return err
	}
	return writer.Flush()
}
