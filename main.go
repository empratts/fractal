package main

import (
	"encoding/csv"
	"fmt"
	"fractal/compute"
	"fractal/gui"
	"image"
	"image/color"
	"image/png"
	"os"
	"strconv"
	"sync"
)

var width, height int = 512, 512      //width and height of the image in pixels
var xRange, yRange float64 = 2.0, 2.0 //the range +/- plotted on the cartesian plane
var exponent float64 = 9.71           //exponent of the mandelbrot function
var iterations int = 10000            //the number of iterations to compute before bailing out
var bailRadius float64 = 200.0        //point to assume the function will diverge
var mbRoutines int = 8                //number of concurrent goroutines to use

type imagePoint struct {
	loc        compute.Point
	pointColor color.RGBA
}

type csvPoint struct {
	loc   compute.Point
	score float64
}

func main() {

	if len(os.Args) > 1 {
		expIn, err := strconv.ParseFloat(os.Args[1], 64)
		if err == nil {
			exponent = expIn
		}
	}

	c := compute.Computer{
		Width:      width,
		Height:     height,
		XRange:     xRange,
		YRange:     yRange,
		Exponent:   exponent,
		Iterations: iterations,
		BailRadius: bailRadius,
		MbRoutines: mbRoutines,
	}

	g := gui.Gui{}

	g.Spawn(c)
}

//mb should return the number of cycles that the mandelbrot function remains inside the bailout circle for the given input
func writeImage(imgChan <-chan imagePoint, wg *sync.WaitGroup) {

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	pixilCount := width * height

	var ip imagePoint

	// Set color for each pixel.
	for i := 0; i < pixilCount; i++ {
		ip = <-imgChan
		img.Set(ip.loc.X, ip.loc.Y, ip.pointColor)
	}

	filename := fmt.Sprintf("exp %v bail %v iter %v - %vx%v-ES.png", exponent, bailRadius, iterations, width, height)
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
	filename := fmt.Sprintf("exp %v bail %v iter %v - %vx%v-ES.csv", exponent, bailRadius, iterations, width, height)
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
					if v.loc.X == i && v.loc.Y == j {
						csvStrings = append(csvStrings, strconv.FormatFloat(v.score, 'f', -1, 64))
						found = true
						buffer = append(buffer[:k], buffer[k+1:]...)
						break
					}
				}
				//if not, wait for something on the channel and place it in the buffer
				if !found {
					pointIn := <-csvChan
					buffer = append(buffer, pointIn)
					for pointIn.loc.X != i || pointIn.loc.Y != j {
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
