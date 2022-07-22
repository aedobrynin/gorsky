package util

import (
    "errors"
    "fmt"
    "golang.org/x/image/draw"
    "golang.org/x/image/tiff"
    "image"
    "image/color"
    "image/jpeg"
    "image/png"
    "os"
    "path/filepath"
    "sync"
    "sync/atomic"
)

func ProcessImages(paths []string, resultDirPath string, maxWorkers int) error {
    resultDirAbsPath, err := createDir(resultDirPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error while creating result directory %v: %v\n", resultDirPath, err)
        return errors.New("some files weren't processed, check stderr for more information")
    }

    var okExecutions uint64

    inputChan := make(chan string, len(paths))
    for _, path := range paths {
        inputChan <- path
    }
    wg := new(sync.WaitGroup)
    for i := 0; i < min(maxWorkers, len(paths)); i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for path := range inputChan {
                if err := processImage(path, resultDirAbsPath); err != nil {
                    fmt.Fprintf(os.Stderr, "Error processing file %v: %v\n", path, err)
                } else {
                    atomic.AddUint64(&okExecutions, 1)
                    fmt.Printf("%v/%v: Successfuly processed %v\n", atomic.LoadUint64(&okExecutions), len(paths), path)
                }
            }
        }()
    }
    close(inputChan)
    wg.Wait()

    fmt.Println("Job is done")
    if okExecutions == uint64(len(paths)) {
        return nil
    }
    return errors.New("some files weren't processed, check stderr for more information")
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
    switch imgData.(type) {
    case *image.Gray, *image.Gray16:
        {
        }
    default:
        return errors.New("wrong image format")
    }

    r, g, b := splitIntoLayers(&imgData)

    getPyramidGoroutine := func(img *image.Gray16, minLayerSize int) <-chan []*image.Gray16 {
        outChan := make(chan []*image.Gray16)
        go func() {
            defer close(outChan)
            outChan <- getPyramid(img, minLayerSize)
        }()
        return outChan
    }

    rPyramidChan := getPyramidGoroutine(r, 100)
    gPyramidChan := getPyramidGoroutine(g, 100)
    bPyramidChan := getPyramidGoroutine(b, 100)

    rPyramid := <-rPyramidChan
    gPyramid := <-gPyramidChan
    bPyramid := <-bPyramidChan

    getBestShiftByPyramidSearchGoroutine := func(stay, shift []*image.Gray16) <-chan int {
        outChan := make(chan int)
        go func() {
            defer close(outChan)
            xShift, yShift := getBestShiftByPyramidSearch(stay, shift)
            outChan <- xShift
            outChan <- yShift
        }()
        return outChan
    }
    rShiftChan := getBestShiftByPyramidSearchGoroutine(gPyramid, rPyramid)
    bShiftChan := getBestShiftByPyramidSearchGoroutine(gPyramid, bPyramid)
    rXShift, rYShift := <-rShiftChan, <-rShiftChan
    bXShift, bYShift := <-bShiftChan, <-bShiftChan

    result := stackLayersWithShifts(r, rXShift, rYShift, g, 0, 0, b, bXShift, bYShift)

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

func splitIntoLayers(img *image.Image) (*image.Gray16, *image.Gray16, *image.Gray16) {
    bounds := (*img).Bounds()
    width, height := bounds.Dx(), bounds.Dy()

    const cutWidthCoeff float64 = 0.1
    cutWidth := int(float64(width) * cutWidthCoeff)

    const cutHeightCoeff float64 = 0.1
    cutHeight := int(float64(height/3) * cutHeightCoeff)

    wg := new(sync.WaitGroup)
    wg.Add(3)

    b := image.NewGray16(image.Rect(0, 0, width-2*cutWidth, height/3-2*cutHeight))
    bCopyRect := image.Rect(cutWidth, cutHeight, width-cutWidth, height/3-cutHeight)
    go func() {
        defer wg.Done()
        draw.Copy(b, image.Pt(0, 0), *img, bCopyRect, draw.Over, nil)
    }()

    g := image.NewGray16(image.Rect(0, 0, width-2*cutWidth, height/3-2*cutHeight))
    gCopyRect := image.Rect(cutWidth, height/3+cutHeight, width-cutWidth, height/3*2-cutHeight)
    go func() {
        defer wg.Done()
        draw.Copy(g, image.Pt(0, 0), *img, gCopyRect, draw.Over, nil)
    }()

    r := image.NewGray16(image.Rect(0, 0, width-2*cutWidth, height/3-2*cutHeight))
    rCopyRect := image.Rect(cutWidth, height/3*2+cutHeight, width-cutWidth, height-cutHeight)
    go func() {
        defer wg.Done()
        draw.Copy(r, image.Pt(0, 0), *img, rCopyRect, draw.Over, nil)
    }()

    wg.Wait()

    return r, g, b
}

func stackLayersWithShifts(r *image.Gray16, rXShift, rYShift int,
g *image.Gray16, gXShift, gYShift int,
b *image.Gray16, bXShift, bYShift int) *image.RGBA64 {
    rBoundsShifted := r.Bounds().Add(image.Pt(rXShift, rYShift))
    gBoundsShifted := g.Bounds().Add(image.Pt(gXShift, gYShift))
    bBoundsShifted := b.Bounds().Add(image.Pt(bXShift, bYShift))
    intersection := rBoundsShifted.Intersect(gBoundsShifted).Intersect(bBoundsShifted)
    width, height := intersection.Dx(), intersection.Dy()
    result := image.NewRGBA64(image.Rect(0, 0, width, height))
    for i := intersection.Min.X; i < intersection.Max.X; i++ {
        for j := intersection.Min.X; j < intersection.Max.Y; j++ {
            rC := (*r).Gray16At(i-rXShift, j-rYShift).Y
            gC := (*g).Gray16At(i-gXShift, j-gYShift).Y
            bC := (*b).Gray16At(i-bXShift, j-bYShift).Y
            result.Set(i-intersection.Min.X, j-intersection.Min.Y, color.RGBA64{R: rC, G: gC, B: bC, A: 255})
        }
    }

    return result
}

func getBestShift(stay, shift *image.Gray16, xSearchRange, ySearchRange [2]int) (int, int) {
    var bestCorrel int64 = 0
    var bestXShift, bestYShift int

    width, height := stay.Bounds().Dx(), stay.Bounds().Dy()

    for xShift := xSearchRange[0]; xShift <= xSearchRange[1]; xShift++ {
        for yShift := ySearchRange[0]; yShift <= ySearchRange[1]; yShift++ {
            var curCorrel int64 = 0
            for i := 0; i < width; i++ {
                for j := 0; j < height; j++ {
                    a := stay.Gray16At(i, j).Y
                    b := shift.Gray16At((i-xShift+width)%width, (j-yShift+height)%height).Y
                    old := curCorrel
                    curCorrel += int64(a) * int64(b)
                    if curCorrel < old {
                        fmt.Println("overflow")
                    }
                }
            }
            if curCorrel > bestCorrel {
                bestCorrel = curCorrel
                bestXShift, bestYShift = xShift, yShift
            }
        }
    }
    return bestXShift, bestYShift
}

func getPyramid(img *image.Gray16, minLayerSize int) []*image.Gray16 {
    pyramid := []*image.Gray16{img}

	curWidth, curHeight := img.Bounds().Dx()/2, img.Bounds().Dy()/2
	for min(curWidth, curHeight) > 100 {
		layer := image.NewGray16(image.Rect(0, 0, curWidth, curHeight))
		draw.NearestNeighbor.Scale(layer, layer.Bounds(), img, img.Bounds(), draw.Over, nil)
		pyramid = append(pyramid, layer)
		curWidth /= 2
		curHeight /= 2
	}
	return pyramid
}

func getBestShiftByPyramidSearch(stay, shift []*image.Gray16) (int, int) {
	xSearchRange := [2]int{-7, 7}
	ySearchRange := [2]int{-7, 7}

	var xShift, yShift int
	for i := len(stay) - 1; i >= 0; i-- {
		xShift, yShift = getBestShift(stay[i], shift[i], xSearchRange, ySearchRange)
		xSearchRange = [2]int{xShift*2 - 2, xShift*2 + 2}
		ySearchRange = [2]int{yShift*2 - 2, yShift*2 + 2}
	}
	return xShift, yShift
}

func max(val0 int, val ...int) int {
	mx := val0
	for _, v := range val {
		if v > mx {
			mx = v
		}
	}
	return mx
}

func min(val0 int, val ...int) int {
	mn := val0
	for _, v := range val {
		if v < mn {
			mn = v
		}
	}
	return mn
}

func abs(val int) int {
	if val > 0 {
		return val
	}
	return -val
}
