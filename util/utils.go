package util

import (
    "fmt"
    "os"
    "sync"
    "errors"
)

func ProcessImages(paths []string, resultDirPath string) {
    fmt.Println("Paths:", paths, "; resultDirPath:", resultDirPath)

    wg := new(sync.WaitGroup)
    wg.Add(len(paths))
    for _, path := range paths {
        go func(p string) {
            processImage(p, resultDirPath)
            wg.Done()
        }(path)
    }
    wg.Wait()

    fmt.Println("Job is done")
}

func processImage(path string, resultDirPath string) {
    exists, err := fileExists(path)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error processing file %s: %s\n", path, err.Error())
        return
    } else if !exists {
        fmt.Fprintf(os.Stderr, "Error processing file %v: file does not exist\n", path)
        return
    }
    fmt.Printf("Ok %s\n", path)
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
