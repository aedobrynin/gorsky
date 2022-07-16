package util

import (
    "fmt"
    "os"
    "sync"
)

func ProcessImages(paths []string, resultDirPath string) {
    fmt.Println("Paths:", paths, "; resultDirPath:", resultDirPath)

    wg := new(sync.WaitGroup)
    wg.Add(len(paths))
    for _, path := range paths {
        go func(p string) {
            ProcessImage(p, resultDirPath)
            wg.Done()
        }(path)
    }
    wg.Wait()
    fmt.Println("Job is done")
}

func ProcessImage(path string, resultDirPath string) {
    fmt.Printf("Working on %s\n", path)
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
