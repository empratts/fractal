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