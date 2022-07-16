package util

import (
    "image"
    _ "image/jpeg"
    "image/png"
    _ "golang.org/x/image/tiff"
    "golang.org/x/image/draw"
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

    imgData, _, err := image.Decode(file)
    if err != nil {
        return err
    }
    splitIntoLayers(&imgData)
    //redLayer, greenLayer, blueLayer := splitIntoLayers(&imgData)
    //fmt.Printf("Ok %v, %v\n", path, err, redLayer, greenLayer, blueLayer)
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

func splitIntoLayers(img *image.Image) (*image.RGBA64, *image.RGBA64, *image.RGBA64) {
    bounds := (*img).Bounds()
    width, height := bounds.Dx(), bounds.Dy()

    b := image.NewRGBA64(image.Rect(0, 0, width, height / 3))
    draw.Copy(b, image.Pt(0, 0), *img, image.Rect(0, 0, width, height / 3), draw.Over, nil)
    b_out, _ := os.Create("b.png")
    defer b_out.Close()
    png.Encode(b_out, b)

    g := image.NewRGBA64(image.Rect(0, 0, width, height / 3))
    draw.Copy(g, image.Pt(0, 0), *img, image.Rect(0, height / 3, width, height / 3 * 2), draw.Over, nil)
    g_out, _ := os.Create("g.png")
    defer g_out.Close()
    png.Encode(g_out, g)

    r := image.NewRGBA64(image.Rect(0, 0, width, height / 3))
    draw.Copy(r, image.Pt(0, 0), *img, image.Rect(0, height / 3 * 2, width, height), draw.Over, nil)
    r_out, _ := os.Create("r.png")
    defer r_out.Close()
    png.Encode(r_out, r)

    return r, g, b
}
