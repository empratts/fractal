package gui

import (
	"fmt"
	"fractal/compute"
	"image"
	"image/color"
	"math"
	"sync"
	"sync/atomic"

	"fyne.io/fyne"
)

type fractalImage struct {
	window            fyne.Window
	canvas            fyne.CanvasObject
	img               *image.RGBA
	width             int
	height            int
	activeCalculation *int32
	computer          *compute.Computer
}

func (f *fractalImage) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	min := 0
	if size.Height < size.Width {
		min = size.Height
	} else {
		min = size.Width
	}
	imgSize := fyne.Size{Width: min, Height: min}
	pos := fyne.NewPos((size.Width-min)/2, (size.Height-min)/2)

	f.canvas.Resize(imgSize)
	f.canvas.Move(pos)
}

func (f *fractalImage) sendImage(points <-chan compute.Point, wg *sync.WaitGroup) {
	count := 0

	for {
		p, more := <-points
		if more {
			count++
			mbScore := float64(p.Score) + p.ExitSpeed

			f.img.Set(p.X, p.Y, getHeatColor(mbScore, p.Iterations))

			if count > 127 {
				f.refresh()
				count = 0
			}
		} else {
			break
		}
	}
	f.refresh()
	wg.Done()

}

func (f *fractalImage) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(f.width, f.height)
}

func (f *fractalImage) refresh() {
	f.window.Canvas().Refresh(f.canvas)
}

func (f *fractalImage) Tapped(p *fyne.PointEvent) {
	fmt.Println("Tapped")
}

func (f *fractalImage) TappedSecondary(p *fyne.PointEvent) {
	fmt.Println("TappedSecondary")
}

func getHeatColor(mbValue float64, iterations int) color.RGBA {
	var colors [][]uint8 = [][]uint8{
		{0, 0, 0},
		{255, 255, 255},
		{255, 255, 255},
		{255, 255, 255},
		{255, 255, 255},
		{255, 255, 255},
		{255, 255, 255},
		{255, 255, 255},
		{70, 70, 255},
		{35, 35, 255},
		{0, 0, 255},
	}

	return getHeatColorWithPalet(mbValue, iterations, colors)
}

func getHeatColorWithPalet(mbValue float64, iterations int, colors [][]uint8) color.RGBA {

	intensity := mbValue / float64(iterations)

	intensity = 1.0 - math.Pow(intensity, .25)

	intensity *= float64(len(colors) - 1)

	i := int(intensity)

	if i == len(colors)-1 {
		return color.RGBA{colors[i][0], colors[i][1], colors[i][2], 0xff}
	}

	ratio := intensity - float64(i)

	r := uint8(float64(colors[i][0])*(1.0-ratio) + float64(colors[i+1][0])*(ratio))
	g := uint8(float64(colors[i][1])*(1.0-ratio) + float64(colors[i+1][1])*(ratio))
	b := uint8(float64(colors[i][2])*(1.0-ratio) + float64(colors[i+1][2])*(ratio))

	return color.RGBA{r, g, b, 0xff}
}

func (f *fractalImage) highlightScore(score int) {

	if atomic.LoadInt32(f.activeCalculation) == 0 {
		fmt.Println("Attempting to highlight scores")
		points := make(chan compute.Point)

		go f.computer.RequestScorePoints(score, points)

		for {
			p, more := <-points

			if more {
				setColor := color.RGBA{R: 255, G: 0, B: 0, A: 255}
				f.img.Set(p.X, p.Y, setColor)
			} else {
				break
			}
		}
	}
	f.refresh()
}

func (f *fractalImage) stopHighlight(score int) {

	if atomic.LoadInt32(f.activeCalculation) == 0 {
		fmt.Println("Attempting to remove highlights")
		points := make(chan compute.Point)

		go f.computer.RequestScorePoints(score, points)

		for {
			p, more := <-points

			if more {
				f.img.Set(p.X, p.Y, getHeatColor(float64(p.Score)+p.ExitSpeed, p.Iterations))
			} else {
				break
			}
		}
	}
	f.refresh()
}
