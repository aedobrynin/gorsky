package util

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
    "image/color"
    "golang.org/x/image/draw"
	"golang.org/x/image/tiff"
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
                fmt.Printf("%v/%v: Successfuly processed %v\n", atomic.LoadUint64(&okExecutions), len(paths), p)
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

	inFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer inFile.Close()

	imgData, imgType, err := image.Decode(inFile)
	if err != nil {
		return err
	}
	r, g, b := splitIntoLayers(&imgData)

    result, err := stackLayers(r, g, b)
    if err != nil {
        return err
    }

    filename := filepath.Base(path)
	resultPath := filepath.Join(resultDirPath, filename)
    outFile, err := os.Create(resultPath)
    if err != nil {
        return err
    }
    defer outFile.Close()

    switch imgType {
    case "jpeg":
        err = jpeg.Encode(outFile, result, nil)
    case "png":
        err = png.Encode(outFile, result)
    case "tiff":
        err = tiff.Encode(outFile, result, nil)
    }
    if err != nil {
        return err
    }

    return nil
}

func createDir(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(absPath, 0777)
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

	const cutWidthCoeff float64 = 0.05
	cutWidth := int(float64(width) * cutWidthCoeff)

	const cutHeightCoeff float64 = 0.03
	cutHeight := int(float64(height / 3) * cutHeightCoeff)

    wg := new(sync.WaitGroup)
    wg.Add(3)

	b := image.NewRGBA64(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    bCopyRect := image.Rect(cutWidth, cutHeight, width - cutWidth, height / 3 - cutHeight)
    go func() {
        draw.Copy(b, image.Pt(0, 0), *img, bCopyRect, draw.Over, nil)
        wg.Done()
    }()

    g := image.NewRGBA64(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    gCopyRect := image.Rect(cutWidth, height / 3 + cutHeight, width - cutWidth, height / 3 * 2 - cutHeight)
	go func() {
        draw.Copy(g, image.Pt(0, 0), *img, gCopyRect, draw.Over, nil)
        wg.Done()
    }()

	r := image.NewRGBA64(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    rCopyRect := image.Rect(cutWidth, height / 3 * 2 + cutHeight, width - cutWidth, height - cutHeight)
	go func() {
        draw.Copy(r, image.Pt(0, 0), *img, rCopyRect, draw.Over, nil)
        wg.Done()
    }()

    wg.Wait()

    return r, g, b
}

func stackLayers(r, g, b *image.RGBA64) (*image.RGBA64, error) {
    if (*r).Bounds() != (*g).Bounds() || (*r).Bounds() != (*b).Bounds() || (*g).Bounds() != (*b).Bounds() {
        return nil, errors.New("Layer sizes do not match")
    }

    result := image.NewRGBA64((*r).Bounds())
    width, height := result.Bounds().Dy(), result.Bounds().Dx()
    for i := 0; i < height; i++ {
        for j := 0; j < width; j++ {
            rC, _, _, _ := (*r).At(i, j).RGBA()
            _, gC, _, _ := (*g).At(i, j).RGBA()
            _, _, bC, _ := (*b).At(i, j).RGBA()
            result.Set(i, j, color.RGBA{R: uint8(rC), G: uint8(gC), B: uint8(bC), A: 255})
        }
    }
    return result, nil
}
