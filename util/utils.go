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

    shiftGoroutine := func(stay, shift *image.Gray16) <-chan int{
       outChan := make(chan int)
        go func() {
            defer close(outChan)
            xShift, yShift := getBestShift(stay, shift)
            outChan <- xShift
            outChan <- yShift
        }()
        return outChan
    }
    gShiftChan := shiftGoroutine(r, g)
    bShiftChan := shiftGoroutine(r, b)
    gXShift, gYShift := <-gShiftChan, <-gShiftChan
    bXShift, bYShift := <-bShiftChan, <-bShiftChan
    result, err := stackLayersWithShifts(r, 0, 0, g, gXShift, gYShift, b, bXShift, bYShift)
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
            rC := (*r).Gray16At(i + abs(rXShift), j + abs(rYShift)).Y
            gC := (*g).Gray16At(i + abs(gXShift), j + abs(gYShift)).Y
            bC := (*b).Gray16At(i + abs(bXShift), j + abs(bYShift)).Y
            result.Set(i, j, color.RGBA64{R: rC, G: gC, B: bC, A: 255})
        }
    }
    return result, nil
}

func getBestShift(stay, shift *image.Gray16) (int, int) {
    var bestCorrel int64 = 0
    var bestXShift, bestYShift int

    width, height := stay.Bounds().Dx(), stay.Bounds().Dy()

    for xShift := -100; xShift <= 100; xShift++ {
        for yShift := -100; yShift <= 100; yShift++ {
            var curCorrel int64 = 0
            for i := max(-xShift, 0); i + xShift < width; i++ {
                for j := max(-yShift, 0); j + yShift < height; j++ {
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

func shiftAndCutEmpty(img *image.Gray16, xShift, yShift int) *image.Gray16 {
    width, height := img.Bounds().Dx(), img.Bounds().Dy()
    cropTo := image.Rect(max(0, xShift), max(0, yShift), min(width + xShift, width), min(height + yShift, height))
    return getCropped(img, cropTo)
}

func getCropped(img *image.Gray16, r image.Rectangle) *image.Gray16 {
    return img.SubImage(r).(*image.Gray16)
}
