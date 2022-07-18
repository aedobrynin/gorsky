/*
    "pprof"
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
    "github.com/aedobrynin/gorsky/cmd"
    "os"
    "log"
    "runtime/pprof"
)

func main() {
    f, err := os.Create("cpu.prof")
    if err != nil {
        log.Fatal("could not create CPU profile: ", err)
    }
    defer f.Close() // error handling omitted for example
    if err := pprof.StartCPUProfile(f); err != nil {
        log.Fatal("could not start CPU profile: ", err)
    }
    defer pprof.StopCPUProfile()

    cmd.Execute()
}
