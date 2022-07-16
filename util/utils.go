package util

import (
    "image"
    _ "image/jpeg"
    _ "image/png"
    _ "golang.org/x/image/tiff"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

func ProcessImages(paths []string, resultDirPath string) error {
	resultDirAbsPath, err := createDir(resultDirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while creating result directory %v: %v\n", resultDirPath, err)
		return errors.New("Some files weren't processed, check stderr for more information")
	}

	var okExecutions uint64

	wg := new(sync.WaitGroup)
	wg.Add(len(paths))
	for _, path := range paths {
		go func(p string) {
			defer wg.Done()
			if err := processImage(p, resultDirAbsPath); err != nil {
				fmt.Fprintf(os.Stderr, "Error processing file %v: %v\n", p, err)
			} else {
				atomic.AddUint64(&okExecutions, 1)
			}
		}(path)
	}
	wg.Wait()

	fmt.Println("Job is done")
	if okExecutions == uint64(len(paths)) {
		return nil
	}
	return errors.New("Some files weren't processed, check stderr for more information")
}

func processImage(path string, resultDirPath string) error {
	exists, err := fileExists(path)
	if err != nil {
		return err
	}
	if !exists {
		return os.ErrNotExist
	}

    //filename := filepath.Base(path)
    //resultPath := filepath.Join(resultDirPath, filename)

    file, err := os.Open(path)
    if err != nil {
        return err
    }
    defer file.Close()

    _, _, err = image.Decode(file)
    fmt.Printf("Ok %v, %v\n", path, err)
    return nil
}

func createDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(absPath, 0666)
	return absPath, err
}

func fileExists(path string) (bool, error) {
	if info, err := os.Stat(path); err == nil {
		return !info.IsDir(), nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}
