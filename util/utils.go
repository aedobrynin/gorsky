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
        return errors.New("some files weren't processed, check stderr for more information")
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
    case *image.Gray, *image.Gray16: {
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

    rPyramid := <- rPyramidChan
    gPyramid := <- gPyramidChan
    bPyramid := <- bPyramidChan
    fmt.Println(len(rPyramid), len(gPyramid), len(bPyramid))

    getBestShiftByPyramidSearchGoroutine := func(stay, shift []*image.Gray16) <-chan int{
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
    fmt.Println(rXShift, rYShift)
    fmt.Println(bXShift, bYShift)
    result, err := stackLayersWithShifts(r, rXShift, rYShift, g, 0, 0, b, bXShift, bYShift)
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

func splitIntoLayers(img *image.Image) (*image.Gray16, *image.Gray16, *image.Gray16) {
    bounds := (*img).Bounds()
    width, height := bounds.Dx(), bounds.Dy()

    const cutWidthCoeff float64 = 0.05
    cutWidth := int(float64(width) * cutWidthCoeff)

    const cutHeightCoeff float64 = 0.03
    cutHeight := int(float64(height / 3) * cutHeightCoeff)

    wg := new(sync.WaitGroup)
    wg.Add(3)

    b := image.NewGray16(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    bCopyRect := image.Rect(cutWidth, cutHeight, width - cutWidth, height / 3 - cutHeight)
    go func() {
        draw.Copy(b, image.Pt(0, 0), *img, bCopyRect, draw.Over, nil)
        wg.Done()
    }()

    g := image.NewGray16(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    gCopyRect := image.Rect(cutWidth, height / 3 + cutHeight, width - cutWidth, height / 3 * 2 - cutHeight)
    go func() {
        draw.Copy(g, image.Pt(0, 0), *img, gCopyRect, draw.Over, nil)
        wg.Done()
    }()

    r := image.NewGray16(image.Rect(0, 0, width - 2 * cutWidth, height / 3 - 2 * cutHeight))
    rCopyRect := image.Rect(cutWidth, height / 3 * 2 + cutHeight, width - cutWidth, height - cutHeight)
    go func() {
        draw.Copy(r, image.Pt(0, 0), *img, rCopyRect, draw.Over, nil)
        wg.Done()
    }()

    wg.Wait()

    return r, g, b
}

func stackLayersWithShifts(r *image.Gray16, rXShift, rYShift int,
                           g *image.Gray16, gXShift, gYShift int,
                           b *image.Gray16, bXShift, bYShift int) (*image.RGBA64, error) {
    width := min(
        r.Bounds().Dx() - abs(rXShift),
        g.Bounds().Dx() - abs(gXShift),
        b.Bounds().Dx() - abs(bXShift),
    )
    height := min(
        r.Bounds().Dy() - abs(rYShift),
        g.Bounds().Dy() - abs(gYShift),
        b.Bounds().Dy() - abs(bYShift),
    )

    result := image.NewRGBA64(image.Rect(0, 0, width, height))
    for i := 0; i < width; i++ {
        for j := 0; j < height; j++ {
            rC := (*r).Gray16At(i + rXShift, j + rYShift).Y
            gC := (*g).Gray16At(i + gXShift, j + gYShift).Y
            bC := (*b).Gray16At(i + bXShift, j + bYShift).Y
            result.Set(i, j, color.RGBA64{R: rC, G: gC, B: bC, A: 255})
        }
    }
    return result, nil
}

func getBestShift(stay, shift *image.Gray16, xSearchRange, ySearchRange [2]int) (int, int) {
    var bestCorrel int64 = 0
    var bestXShift, bestYShift int

    width, height := stay.Bounds().Dx(), stay.Bounds().Dy()

    for xShift := xSearchRange[0]; xShift <= xSearchRange[1]; xShift++ {
        for yShift := ySearchRange[0]; yShift <= ySearchRange[1]; yShift++ {
            var curCorrel int64 = 0
            for i := 0; i + xShift < width; i++ {
                for j := 0; j + yShift < height; j++ {
                    a := stay.Gray16At(i, j).Y
                    b := shift.Gray16At(i + xShift, j + yShift).Y
                    curCorrel += int64(a) * int64(b)
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
    pyramid := []*image.Gray16{img} // TODO: calculate layers count using log2

    cur_width, cur_height := img.Bounds().Dx() / 2, img.Bounds().Dy() / 2
    for min(cur_width, cur_height) > 100 {
        layer := image.NewGray16(image.Rect(0, 0, cur_width, cur_height))
        draw.BiLinear.Scale(layer, layer.Bounds(), img, img.Bounds(), draw.Over, nil)
        pyramid = append(pyramid, layer)
        cur_width /= 2
        cur_height /= 2
    }
    return pyramid
}

func getBestShiftByPyramidSearch(stay, shift []*image.Gray16) (int, int) {
    xSearchRange := [2]int{-30, 30}
    ySearchRange := [2]int{-30, 30}

    var xShift, yShift int
    for i := len(stay) - 1; i >= 0; i-- {
        xShift, yShift = getBestShift(stay[i], shift[i], xSearchRange, ySearchRange)
        xSearchRange = [2]int{xShift * 2 - 2, xShift * 2 + 2}
        ySearchRange = [2]int{yShift * 2 - 2, yShift * 2 + 2}
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
