// Copyright (C) 2026 InterfaceKentar 
// Licensed under the GNU Public License version 3.0 or later.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const MaxBufferSize int64 = 10 * 1024 * 1024 // 10MB
var (
	LicenseText string = ""
	err         error
)

func main() {
	var (
		rootPath    *string
		licensePath *string
		ext         *string
	)
	rootPath    = flag.String("p", "path", "The root directory of the Java project")
	licensePath = flag.String("l", "license", "The file location of the license text.")
	ext         = flag.String("e", "extention", "define what file types to apply append.")

	flag.Parse()

	if *rootPath == "" {
		fmt.Println("Error: Please provide a project path using -path")
		os.Exit(1)
	}
	if *licensePath == "" {
		fmt.Println("Error: Please provide a license path using -license")
		os.Exit(1)
	}
	if *ext == "" {
		fmt.Println("Error: Please provide which file type to modify.")
		os.Exit(1)
	}

	var extSet *map[string]bool
	extSet, err = createExtensionMap(*ext)
	if err != nil {
		log.Fatalf("Failed to load extension set from cli arguments.%v\n", err)
		os.Exit(1)
	}

	var licenseFile *os.File
	licenseFile, err = os.Open(*licensePath)
	if err != nil {
		log.Fatalf("Failed to open a license file. %v\n", err)
		os.Exit(1)
	}
	defer licenseFile.Close()

	LicenseText, err = loadLicenseText(licenseFile)
	if err != nil {
		log.Fatalf("Failed to get a license text from license file. %v\n", err)
		os.Exit(1)
	}

	var (
		info os.FileInfo
		err  error
	)
	info, err = os.Stat(*rootPath)

	if os.IsNotExist(err) {
		fmt.Printf("Error: The path '%s' does not exist.\n", *rootPath)
		os.Exit(1)
	}

	var files []string
	files, err = findExtension(*rootPath, extSet)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	var path string
	for _, path = range files {
		err = prependLicense(path, &info)
		if err != nil {
			fmt.Printf("Failed to process [%s]: %v\n", path, err)
			continue
		}
		fmt.Printf("Successfully updated: %s\n", path)
	}
}

func createExtensionMap(str string) (*map[string]bool, error) {
	if str == "" {
		return nil, err
	}
	var (
		extSet  map[string]bool = make(map[string]bool)
		splitedString []string        = strings.Split(str, ",")
	)
	for i := range splitedString {
		extSet["." + splitedString[i]] = true
	}
	return &extSet, nil
}

func loadLicenseText(file *os.File) (string, error) {
	var (
		reader   *bufio.Reader = bufio.NewReader(file)
		contents []byte
	)
	for {
		var buffer []byte
		buffer, err = reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to get a license text from a flie. %v\n", err)
			return "", err
		}
		contents = slices.Concat(contents, buffer)
	}
	return string(contents), nil
}

func findExtension(root string, extSet *map[string]bool) ([]string, error) {
	var files []string
	var err error
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (*extSet)[filepath.Ext(path)] == true {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func prependLicense(filePath string, originalMode *os.FileInfo) error {
	var (
		file_ori *os.File
		err      error
		info     os.FileInfo
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
