package create

import (
	"time"

	"github.com/schollz/progressbar/v3"
)

var bar *progressbar.ProgressBar
var totalSteps int

/**
* ProgressAdd: creates the progress bar if needed and adds step to the total number of steps.
* @param step int
**/
func ProgressAdd(step int) {
	if bar == nil {
		bar = progressbar.Default(100)
	}

	totalSteps += step
}

/**
* ProgressNext: advances the bar one step (100/totalSteps) with a short pause so it is visible.
**/
func ProgressNext() {
	if bar == nil {
		bar = progressbar.Default(100)
		totalSteps = 100
	}

	bar.Add(100 / totalSteps)
	time.Sleep(40 * time.Millisecond)
}
