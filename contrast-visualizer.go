package main

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

var calcColorLuminanceCache = map[float64]float64{}

func calcColorLuminance(rgb float64) float64 {
	cache, has := calcColorLuminanceCache[rgb]
	if has {
		return cache
	}
	if rgb <= 0.03928 {
		cache = rgb / 12.92
	} else {
		cache = math.Pow(((rgb + 0.055) / 1.055), 2.4)
	}
	calcColorLuminanceCache[rgb] = cache
	return cache
}

var calcRelativeLuminanceCache = map[color.Color]float64{}

// https://waic.jp/translations/WCAG20/Overview.html#relativeluminancedef
func calcRelativeLuminance(color color.Color) float64 {
	cache, has := calcRelativeLuminanceCache[color]
	if has {
		return cache
	}
	r8, g8, b8, _ := color.RGBA()
	rs, gs, bs := float64(r8)/65535, float64(g8)/65535, float64(b8)/65535
	r, g, b := calcColorLuminance(rs), calcColorLuminance(gs), calcColorLuminance(bs)
	cache = 0.2126*r + 0.7152*g + 0.0722*b
	calcRelativeLuminanceCache[color] = cache
	return cache
}

// https://waic.jp/translations/WCAG20/Overview.html#contrast-ratiodef
func calcContrastRatio(color1 color.Color, color2 color.Color) float64 {
	l1, l2 := calcRelativeLuminance(color1), calcRelativeLuminance(color2)

	if l1 > l2 {
		return (l1 + 0.05) / (l2 + 0.05)
	}
	return (l2 + 0.05) / (l1 + 0.05)
}

func calcAverageContrastRatio(radius int, img image.Image, point image.Point) float64 {
	aroundPoints := make([]image.Point, (radius*2+1)*(radius*2+1)-1)
	for dx := -radius; dx <= radius; dx++ {
		for dy := -radius; dy <= radius; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			dPoint := image.Point{X: point.X + dx, Y: point.Y + dy}
			if dPoint.In(img.Bounds()) {
				aroundPoints = append(aroundPoints, dPoint)
			}
		}
	}

	rSum, gSum, bSum := uint32(0), uint32(0), uint32(0)
	for _, comparePoint := range aroundPoints {
		r, g, b, _ := img.At(comparePoint.X, comparePoint.Y).RGBA()
		// log.Printf("color1: %v, color2: {%d %d %d}\n", img.At(point.X, point.Y), r*255/65535, g*255/65535, b*255/65535)
		rSum += r * 255 / 65535
		gSum += g * 255 / 65535
		bSum += b * 255 / 65535
	}
	rSum, gSum, bSum = rSum/uint32(len(aroundPoints)), gSum/uint32(len(aroundPoints)), bSum/uint32(len(aroundPoints))

	color := color.RGBA{
		R: uint8(rSum),
		G: uint8(gSum),
		B: uint8(bSum),
		A: 0,
	}
	// log.Printf("color1: %v, color2: %v\n", img.At(point.X, point.Y), color)

	return calcContrastRatio(img.At(point.X, point.Y), color)

	// sum := float64(0)
	// for _, comparePoint := range aroundPoints {
	// 	sum += calcContrastRatio(img.At(point.X, point.Y), img.At(comparePoint.X, comparePoint.Y))
	// }
	// return sum / float64(len(aroundPoints))
}

func converFloatToGrayScale(contrastRatio float64) (color.Gray, error) {
	if contrastRatio < 1 {
		return color.Gray{}, errors.New("contrastRatio should be more than 0")
	}
	if contrastRatio > 21 {
		return color.Gray{}, errors.New("contrastRatio should be less than 1")
	}
	// var scale float64
	// i1, i2, i3 := float64(30), float64(30), float64(50)
	// if contrastRatio < 3 {
	// 	scale = 255 - ((contrastRatio-1)/2)*i1
	// } else if contrastRatio < 4.5 {
	// 	scale = 255 - (i1 + ((contrastRatio-3)/1.5)*i2)
	// } else if contrastRatio < 7 {
	// 	scale = 255 - (i1 + i2 + ((contrastRatio-4.5)/2.5)*i3)
	// } else {
	// 	scale = 255 - (i1 + i2 + i3 + ((contrastRatio-7)/14)*(255-i1-i2-i3))
	// }

	// return color.Gray{
	// 	Y: uint8(scale),
	// }, nil
	return color.Gray{
		Y: uint8(math.Pow(((contrastRatio-1)/20), float64(1)/4) * 255),
	}, nil
}

func main() {
	filePath := os.Args[1]
	// radius, err := strconv.Atoi(os.Args[2])
	radius := 1
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	src, err := png.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	img := image.NewGray(
		image.Rectangle{
			src.Bounds().Min,
			src.Bounds().Max,
		},
	)

	bar := progressbar.Default(int64(src.Bounds().Dx() * src.Bounds().Dy()))
	for x := 0; x < src.Bounds().Dx(); x++ {
		for y := 0; y < src.Bounds().Dy(); y++ {
			bar.Add(1)
			gray, err := converFloatToGrayScale(calcAverageContrastRatio(radius, src, image.Point{X: x, Y: y}))
			if err != nil {
				log.Fatal(err)
			}
			img.SetGray(x, y, gray)
		}
	}

	outPath := path.Join(filepath.Dir(filePath), "contrast-ratio-"+filepath.Base(filePath))

	file, err = os.Create(outPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		log.Fatal(err)
	}
}
