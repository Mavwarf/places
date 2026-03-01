// mkicon generates a 256x256 PNG app icon: colored circle with white "P".
package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

func main() {
	const size = 256
	const cx, cy = size / 2, size / 2
	const radius = size/2 - 4

	accent := color.RGBA{122, 162, 247, 255} // #7aa2f7 — Tokyo Night blue
	white := color.RGBA{255, 255, 255, 255}

	img := image.NewRGBA(image.Rect(0, 0, size, size))

	// Draw filled circle with anti-aliased edges.
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float64(x) - float64(cx) + 0.5
			dy := float64(y) - float64(cy) + 0.5
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= float64(radius)-0.5 {
				img.Set(x, y, accent)
			} else if dist <= float64(radius)+0.5 {
				// Anti-alias edge.
				a := float64(radius) + 0.5 - dist
				img.Set(x, y, color.RGBA{accent.R, accent.G, accent.B, uint8(a * 255)})
			}
		}
	}

	// Draw "P" letter using simple filled rectangles.
	// The P consists of a vertical bar and a curved top bowl.
	drawP(img, cx, cy, white)

	path := "appicon.png"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

func drawP(img *image.RGBA, cx, cy int, col color.RGBA) {
	// Letter dimensions relative to 256px canvas.
	// P: vertical stem + semicircular bowl on the upper half.

	stemLeft := cx - 36
	stemRight := cx - 12
	stemTop := cy - 60
	stemBottom := cy + 68

	// Vertical stem.
	fillRect(img, stemLeft, stemTop, stemRight, stemBottom, col)

	// Top horizontal bar of the bowl.
	fillRect(img, stemRight, stemTop, cx+30, stemTop+24, col)

	// Bottom horizontal bar of the bowl.
	bowlMidY := cy - 2
	fillRect(img, stemRight, bowlMidY, cx+30, bowlMidY+24, col)

	// Right curve of the bowl (semicircle).
	bowlCx := float64(cx + 30)
	bowlCy := float64(stemTop+12+bowlMidY+12) / 2
	bowlR := float64(bowlMidY+12-stemTop-12) / 2

	for y := stemTop; y <= bowlMidY+24; y++ {
		for x := cx + 10; x < cx+70; x++ {
			dx := float64(x) - bowlCx
			dy := float64(y) - bowlCy
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist <= bowlR+12 && dist >= bowlR-12 && dx >= 0 {
				img.Set(x, y, col)
			}
		}
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.Set(x, y, col)
		}
	}
}
