package gui

import (
	"fmt"
	"fractal/compute"
	"image"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/canvas"
	"fyne.io/fyne/widget"
)

type Gui struct {
	app                   fyne.App
	win                   fyne.Window
	c                     compute.Computer
	image                 *fractalImage
	activeImage           int32
	activeCalculation     int32
	exponentSlider        *widget.Slider
	exponentEntry         *widget.Entry
	updateButtonContainer *scoreUpdate
	calculateButton       *widget.Button
}

func (g *Gui) Spawn(cin compute.Computer) {
	g.c = cin
	g.app = app.New()

	g.win = g.app.NewWindow("Fractal")
	g.win.SetPadded(false)

	g.image = &fractalImage{window: g.win,
		width:             g.c.Width,
		height:            g.c.Height,
		activeCalculation: &g.activeCalculation,
	}

	upLeft := image.Point{0, 0}
	lowRight := image.Point{g.c.Width - 1, g.c.Height - 1}

	g.image.img = image.NewRGBA(image.Rectangle{upLeft, lowRight})

	g.image.canvas = canvas.NewRasterFromImage(g.image.img)
	g.image.computer = &g.c

	g.activeImage = 0
	g.activeCalculation = 0

	g.exponentEntry = widget.NewEntry()
	g.exponentEntry.Disable()
	//g.exponentEntry.OnChanged = g.updateExponentFromEntry

	g.exponentSlider = widget.NewSlider(1.0, 10.0)
	g.exponentSlider.Step = .01
	g.exponentSlider.Value = g.c.Exponent
	g.updateExponentFromSlider(g.exponentSlider.Value)
	g.exponentSlider.OnChanged = g.updateExponentFromSlider

	g.calculateButton = widget.NewButton("Generate", g.calculateImageLauncher)
	g.updateButtonContainer = newScoreUpdate(10000, g.image)

	g.win.SetContent(widget.NewVBox(
		widget.NewLabel("Fractal Generator"),
		g.calculateButton,
		g.exponentEntry,
		g.exponentSlider,
		widget.NewHBox(fyne.NewContainerWithLayout(g.image, g.image.canvas), g.updateButtonContainer.Display()),
		widget.NewButton("Add", g.updateButtonContainer.AddScoreButton),
	))

	g.win.ShowAndRun()
}

func test(b bool) {
	fmt.Println("Test", b)
}

func (g *Gui) calculateImageLauncher() {
	if atomic.CompareAndSwapInt32(&g.activeCalculation, 0, 1) {
		go g.calculateImage()
	}
}

func (g *Gui) calculateImage() {

	//change this to have a popup asking if you want to overwrite the active image
	if atomic.LoadInt32(&g.activeImage) == 1 {
		fmt.Println("Overwriting Image")
	}

	pc := make(chan compute.Point)

	g.c.Exponent = g.exponentSlider.Value

	var wg sync.WaitGroup

	wg.Add(1)
	go g.c.Calculate(pc, &wg)

	wg.Add(1)
	go g.image.sendImage(pc, &wg)

	wg.Wait()

	atomic.StoreInt32(&g.activeImage, 1)
	atomic.StoreInt32(&g.activeCalculation, 0)
}

func (g *Gui) recolorImage() {
	//TODO: Move most of this to fractalImage

	//if there is not an active image to recolor, do nothing
	if atomic.LoadInt32(&g.activeImage) == 0 || atomic.LoadInt32(&g.activeCalculation) == 1 {
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

func (g *Gui) updateExponentFromSlider(value float64) {
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
