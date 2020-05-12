package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
	"sync"

	"github.com/superhawk610/bar"
)

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

var width, height int = 256, 256      //width and height of the image in pixels
var xRange, yRange float64 = 2.0, 2.0 //the range +/- plotted on the cartesian plane
var exponent float64 = 3.40           //exponent of the mandelbrot function
var iterations int = 10000            //the number of iterations to compute before bailing out
var bailRadius float64 = 200.0        //point to assume the function will diverge
var mbRoutines int = 8                //number of concurrent goroutines to use

type point struct {
	x int
	y int
}

type imagePoint struct {
	loc        point
	pointColor color.RGBA
}

type csvPoint struct {
	loc   point
	score int
}

func main() {

	if len(os.Args) > 1 {
		expIn, err := strconv.ParseFloat(os.Args[1], 64)
		if err == nil {
			exponent = expIn
		}
	}

	b := bar.New(width)
	var wg sync.WaitGroup
	pChan := make(chan point)
	imgChan := make(chan imagePoint)
	csvChan := make(chan csvPoint)

	for i := 0; i < mbRoutines; i++ {
		go mbConc(pChan, imgChan, csvChan)
	}

	wg.Add(1)
	go writeImage(imgChan, &wg)
	wg.Add(1)
	go writeCSV(csvChan, &wg)

	// Set color for each pixel.
	for xIter := 0; xIter < width; xIter++ {
		for yIter := 0; yIter < height; yIter++ {
			pChan <- point{x: xIter, y: yIter}
		}
		b.Tick()
	}

	wg.Wait()
}

//mb should return the number of cycles that the mandelbrot function remains inside the bailout circle for the given input
func mb(p point) int {
	//the function takes the form (X^exp + (a + bi))

	//first calculate the starting point for the pixel in the plane between +/- xbound, ybound
	a := ((2.0 * float64(p.x) / (float64(width - 1))) - 1.0) * xRange
	b := ((2.0 * float64(p.y) / (float64(height - 1))) - 1.0) * yRange * -1.0 //the -1 accounts for the fact that the y coordinates in images count from the top down, not the bottom up

	xa := 0.0
	xb := 0.0

	for i := 0; i < iterations; i++ {
		//repeated application of MB
		xa, xb = complexPower(xa, xb, exponent)
		xa += a
		xb += b
		//check if the result stays inside the bailout circle
		bailDistance := math.Sqrt(xa*xa + xb*xb)
		if bailDistance > bailRadius {
			return i
		}
	}
	return iterations
}

func mbConc(pChan <-chan point, imgChan chan<- imagePoint, csvChan chan<- csvPoint) {
	var p point

	for true {
		p = <-pChan
		output := mb(p)
		newColor := getHeatColor(output, iterations)
		ip := imagePoint{loc: p, pointColor: newColor}
		csv := csvPoint{loc: p, score: output}
		imgChan <- ip
		csvChan <- csv
	}
}

func writeImage(imgChan <-chan imagePoint, wg *sync.WaitGroup) {

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	pixilCount := width * height

	var ip imagePoint

	// Set color for each pixel.
	for i := 0; i < pixilCount; i++ {
		ip = <-imgChan
		img.Set(ip.loc.x, ip.loc.y, ip.pointColor)
	}

	filename := fmt.Sprintf("exp %v bail %v iter %v - %vx%v.png", exponent, bailRadius, iterations, width, height)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("error creating PNG file")
	}
	defer f.Close()
	err = png.Encode(f, img)
	if err != nil {
		fmt.Println("error writing to PNG file")
	}
	wg.Done()
}

func writeCSV(csvChan <-chan csvPoint, wg *sync.WaitGroup) {
	filename := fmt.Sprintf("exp %v bail %v iter %v - %vx%v.csv", exponent, bailRadius, iterations, width, height)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println("error creating CSV file")
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	defer writer.Flush()

	buffer := []csvPoint{}

	for i := 0; i < width; i++ {
		csvStrings := []string{}
		for j := 0; j < height; j++ {
			found := false
			for !found {
				//if you have i,j in the buffer, append it to the slice
				for k, v := range buffer {
					if v.loc.x == i && v.loc.y == j {
						csvStrings = append(csvStrings, strconv.FormatInt(int64(v.score), 10))
						found = true
						buffer = append(buffer[:k], buffer[k+1:]...)
						break
					}
				}
				//if not, wait for something on the channel and place it in the buffer
				if !found {
					pointIn := <-csvChan
					buffer = append(buffer, pointIn)
					for pointIn.loc.x != i || pointIn.loc.y != j {
						pointIn = <-csvChan
						buffer = append(buffer, pointIn)
					}
				}
			}
		}
		//write csvStrings to the file
		err = writer.Write(csvStrings)
		writer.Flush()
		if err != nil {
			fmt.Println("error writing to CSV file")
		}
	}

	wg.Done()
}

func complexPower(a, b, exp float64) (float64, float64) {
	r := math.Sqrt(a*a + b*b)
	theta := math.Atan2(b, a)

	r = math.Pow(r, exp)
	theta = exp * theta

	bRet, aRet := math.Sincos(theta)
	aRet *= r
	bRet *= r

	return aRet, bRet
}

func getHeatColor(mbValue, iterations int) color.RGBA {
	intensity := float64(mbValue) / float64(iterations)

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
