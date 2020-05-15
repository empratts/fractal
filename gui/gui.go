package gui

import (
	"fmt"
	"fractal/compute"
	"image"
	"image/color"
	"math"
	"math/rand"
	"strconv"
	"sync"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/widget"
)

type Gui struct {
	app            fyne.App
	win            fyne.Window
	c              compute.Computer
	image          *fractalImage
	activeImage    bool
	exponentSlider *widget.Slider
	exponentEntry  *widget.Entry
	updateButton   *widget.Button
}

func (g Gui) Spawn(cin compute.Computer) {
	g.c = cin
	g.app = app.New()

	g.win = g.app.NewWindow("Fractal")
	g.win.SetPadded(false)

	g.image = &fractalImage{window: g.win,
		width:  g.c.Width,
		height: g.c.Height,
	}

	upLeft := image.Point{0, 0}
	lowRight := image.Point{g.c.Width - 1, g.c.Height - 1}

	g.image.img = image.NewRGBA(image.Rectangle{upLeft, lowRight})

	g.image.canvas = canvas.NewRasterFromImage(g.image.img)
	g.activeImage = false

	g.exponentEntry = widget.NewEntry()
	g.exponentEntry.Disable()
	//g.exponentEntry.OnChanged = g.updateExponentFromEntry

	g.exponentSlider = widget.NewSlider(1.0, 10.0)
	g.exponentSlider.Step = .01
	g.exponentSlider.Value = g.c.Exponent
	g.updateExponentFromSlider(g.exponentSlider.Value)
	g.exponentSlider.OnChanged = g.updateExponentFromSlider

	g.updateButton = widget.NewButton("10k", g.recolor10k)

	g.win.SetContent(widget.NewVBox(
		widget.NewLabel("Fractal Generator"),
		widget.NewButton("Generate", g.calculateImage),
		g.exponentEntry,
		g.exponentSlider,
		widget.NewHBox(fyne.NewContainerWithLayout(g.image, g.image.canvas), g.updateButton),
		widget.NewButton("Recolor", g.recolorImage),
	))

	g.win.ShowAndRun()
}

func (g Gui) calculateImage() {

	//change this to have a popup asking if you want to overwrite the active image
	if g.activeImage {
		fmt.Println("Overwriting Image")
	}

	g.activeImage = true

	pc := make(chan compute.Point)

	g.c.Exponent = g.exponentSlider.Value

	var wg sync.WaitGroup

	wg.Add(1)
	go g.c.Calculate(pc, &wg)

	count := 0

	for {
		p, more := <-pc
		if more {
			count++
			mbScore := float64(p.Score) + p.ExitSpeed

			g.image.img.Set(p.X, p.Y, getHeatColor(mbScore, g.c.Iterations))

			if count > 127 {
				g.image.refresh()
				count = 0
			}
		} else {
			break
		}
	}
	g.image.refresh()
	wg.Wait()
}

func (g Gui) recolor10k() {

}

type fractalImage struct {
	window fyne.Window
	canvas fyne.CanvasObject
	img    *image.RGBA
	width  int
	height int
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

func (f *fractalImage) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(f.width, f.height)
}

func (f *fractalImage) refresh() {
	f.window.Canvas().Refresh(f.canvas)
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

func (g Gui) recolorImage() {

	//if there is not an active image to recolor, do nothing
	if !g.activeImage {
		return
	}

	var colors [][]uint8 = [][]uint8{{0, 0, 0}}
	var c []uint8

	for i := 0; i < 10; i++ {
		for j := 0; j < 8; j++ {
			c = append(c, uint8(rand.Intn(256)))
		}
		colors = append(colors, c)
	}

	for i := 0; i < g.c.Width; i++ {
		for j := 0; j < g.c.Width; j++ {
			mbScore := g.c.CombinedScores[i][j]
			g.image.img.Set(i, j, getHeatColorWithPalet(mbScore, g.c.Iterations, colors))
		}
	}
	g.image.refresh()
}

func (g Gui) updateExponentFromSlider(value float64) {
	g.exponentEntry.Text = strconv.FormatFloat(value, 'f', 2, 64)
	g.exponentEntry.Refresh()
}

// func (g Gui) updateExponentFromEntry(value string) {
// 	val, err := strconv.ParseFloat(value, 64)

// 	if err != nil || val < 1.0 {
// 		g.exponentEntry.Text = strconv.FormatFloat(g.exponentSlider.Value, 'f', 2, 64)
// 	}

// 	g.exponentSlider.Value = val
// 	g.exponentSlider.Refresh()
// }
