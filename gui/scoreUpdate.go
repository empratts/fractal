package gui

import (
	"fmt"
	"strconv"

	"fyne.io/fyne"
	"fyne.io/fyne/widget"
)

type scoreUpdate struct {
	outerBox    *widget.Box
	innerBoxes  []*widget.Box
	button      []*widget.Button
	check       []*widget.Check
	fi          *fractalImage
	maxScore    int
	buttonCount int
	padding     int
}

func newScoreUpdate(maxScore int, image *fractalImage) *scoreUpdate {
	su := &scoreUpdate{maxScore: maxScore, fi: image}

	su.button = append(su.button, widget.NewButton(strconv.FormatInt(int64(maxScore), 10), su.makeChooseScoreColor(maxScore)))
	su.check = append(su.check, widget.NewCheck("", su.makeSetHighlightState(maxScore)))
	su.innerBoxes = append(su.innerBoxes, widget.NewHBox(su.button[0], su.check[0]))
	su.outerBox = widget.NewVBox(su.innerBoxes[0])

	su.buttonCount = 1
	su.padding = 5

	return su
}

func (s *scoreUpdate) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for k, v := range s.innerBoxes {
		v.Resize(fyne.NewSize(100, 30))
		v.Move(fyne.NewPos(0, k*(s.padding+30)))
	}
}

func (s *scoreUpdate) MinSize(objects []fyne.CanvasObject) fyne.Size {
	width := 100
	height := 30*s.buttonCount + s.padding*(s.buttonCount-1)

	return fyne.NewSize(width, height)
}

func (s *scoreUpdate) makeSetHighlightState(score int) func(bool) {
	return func(state bool) {
		if state {
			go s.fi.highlightScore(score)
			s.outerBox.Refresh()
		} else {
			go s.fi.stopHighlight(score)
			s.outerBox.Refresh()
		}
	}
}

func (s *scoreUpdate) makeChooseScoreColor(score int) func() {
	return func() {
		fmt.Println("Choose score color for:", score)
	}
}

func (s *scoreUpdate) AddScoreButton() {
	s.buttonCount++
	b := widget.NewButton(strconv.FormatInt(int64(100), 10), s.makeChooseScoreColor(100))
	c := widget.NewCheck("", s.makeSetHighlightState(100))
	w := widget.NewHBox(b, c)

	s.innerBoxes = append(s.innerBoxes, w)
	s.outerBox.Append(w)
	s.outerBox.Refresh()
	fmt.Println("added box")
}

func (s *scoreUpdate) Display() *fyne.Container {
	return fyne.NewContainerWithLayout(s, s.outerBox)
}
