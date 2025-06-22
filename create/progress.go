package create

import (
	"time"

	"github.com/schollz/progressbar/v3"
)

var bar *progressbar.ProgressBar
var totalSteps int
var currentStep int

func ProgressAdd(step int) {
	if bar == nil {
		bar = progressbar.Default(100)
	}

	totalSteps += step
}

func ProgressNext() {
	if bar == nil {
		bar = progressbar.Default(100)
		totalSteps = 100
		currentStep = 0
	}

	currentStep++
	bar.Add(100 / totalSteps * currentStep)
	time.Sleep(40 * time.Millisecond)
}
