package compute

import (
	"math"
	"sync"
)

//Computer provides the object that will calculate the "scores" of pixles in a fractal
type Computer struct {
	Width, Height  int         //width and height of the image in pixels
	XRange, YRange float64     //the range +/- plotted on the cartesian plane
	Exponent       float64     //exponent of the mandelbrot function
	Iterations     int         //the number of iterations to compute before bailing out
	BailRadius     float64     //point to assume the function will diverge
	MbRoutines     int         //number of concurrent goroutines to use
	CombinedScores [][]float64 //storage of all scores + exitSpeeds
	ScoreMap       map[int]int //keeps track of how many of each score is in the image
}

//Point represents a single pixil in a fractal
type Point struct {
	X         int
	Y         int
	Score     int
	ExitSpeed float64
}

//Calculate sends all fractal results to the channel passed. No gaurentee is made that the will be in order
func (c Computer) Calculate(points chan<- Point, wg *sync.WaitGroup) {

	if c.CombinedScores == nil || len(c.CombinedScores) != c.Width || len(c.CombinedScores[0]) != c.Height {
		c.CombinedScores = make([][]float64, c.Width)
		for i := 0; i < c.Width; i++ {
			c.CombinedScores[i] = make([]float64, c.Height)
		}
	}

	c.ScoreMap = make(map[int]int)

	var calcWG sync.WaitGroup

	input := make(chan Point)
	record := make(chan Point)

	for i := 0; i < c.MbRoutines; i++ {
		calcWG.Add(1)
		go c.mbConc(input, points, record, &calcWG)
	}

	calcWG.Add(1)
	go c.recordPoints(record, &calcWG)

	for xIter := 0; xIter < c.Width; xIter++ {
		for yIter := 0; yIter < c.Height; yIter++ {
			input <- Point{X: xIter, Y: yIter}
		}
	}

	close(input)
	calcWG.Wait()
	close(points)
	wg.Done()
}

func (c Computer) mbConc(input <-chan Point, solved chan<- Point, record chan<- Point, wg *sync.WaitGroup) {

	for {
		p, more := <-input
		if more {
			p.Score, p.ExitSpeed = c.mb(p)
			solved <- p
			record <- p
		} else {
			wg.Done()
			return
		}
	}
}

//this function records completed points in both CombinedScores and ScoreMap
func (c Computer) recordPoints(record <-chan Point, wg *sync.WaitGroup) {
	for {
		p, chanOk := <-record
		if chanOk {
			c.CombinedScores[p.X][p.Y] = float64(p.Score) + p.ExitSpeed
			scoreCount, mapOK := c.ScoreMap[p.Score]
			if mapOK {
				c.ScoreMap[p.Score] = scoreCount + 1
			} else {
				c.ScoreMap[p.Score] = 1
			}
		} else {
			wg.Done()
			return
		}
	}

}

func (c Computer) mb(p Point) (int, float64) {
	//the function takes the form (X^exp + (a + bi))

	//first calculate the starting point for the pixel in the plane between +/- xbound, ybound
	a := ((2.0 * float64(p.X) / (float64(c.Width - 1))) - 1.0) * c.XRange
	b := ((2.0 * float64(p.Y) / (float64(c.Height - 1))) - 1.0) * c.YRange * -1.0 //the -1 accounts for the fact that the y coordinates in images count from the top down, not the bottom up

	xa := 0.0
	xb := 0.0
	var xaPrev, xbPrev float64

	for i := 0; i < c.Iterations; i++ {
		//repeated application of MB
		xaPrev, xbPrev = xa, xb
		xa, xb = complexPower(xa, xb, c.Exponent)
		xa += a
		xb += b
		//check if the result stays inside the bailout circle
		bailDistance := math.Sqrt(xa*xa + xb*xb)
		if bailDistance > c.BailRadius {

			return i, exitSpeed(xa, xb, xaPrev, xbPrev, c.BailRadius)
		}
	}
	return c.Iterations, 0.0
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

//exitSpeed takes the points before and after exiting the bailout circle. It returns the fraction of the line segment between the
//two points that is inside of the bailout circle. This value is a reflection of how close the point was to exiting one iteration sooner or later
func exitSpeed(x, y, xPrev, yPrev, bailRadius float64) float64 {

	var x1, x2, y1, y2 float64
	exitDistance := math.Sqrt((xPrev-x)*(xPrev-x) + (yPrev-y)*(yPrev-y))

	//min exit speed is .001 to prevent rounding errors caused by subtracting floats with different orders of magnitude
	if bailRadius/exitDistance < .001 {
		return .001
	}

	if x == math.Min(x, xPrev) {
		x1, y1 = x, y
		x2, y2 = xPrev, yPrev
	} else {
		x2, y2 = x, y
		x1, y1 = xPrev, yPrev
	}

	m := (y2 - y1) / (x2 - x1)
	b := y1 - (m * x1)

	aQuad := m*m + 1
	bQuad := 2 * m * b
	cQuad := b*b - bailRadius*bailRadius

	xQuad := (-1.0*bQuad + math.Sqrt(bQuad*bQuad-4*aQuad*cQuad)) / (2.0 * aQuad)

	if xQuad > x2 || xQuad < x1 {
		xQuad = (-1.0*bQuad - math.Sqrt(bQuad*bQuad-4*aQuad*cQuad)) / (2.0 * aQuad)
	}

	yQuad := m*xQuad + b

	interceptDistance := math.Sqrt((xPrev-xQuad)*(xPrev-xQuad) + (yPrev-yQuad)*(yPrev-yQuad))

	exitSpeed := interceptDistance / exitDistance

	return exitSpeed
}
