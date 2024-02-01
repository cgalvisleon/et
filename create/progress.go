package create

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

var bar *progressbar.ProgressBar

func ProgressInit() *progressbar.ProgressBar {
	fmt.Println("")

	if bar == nil {
		bar = progressbar.Default(100)
	}

	return bar
}

func ProgressNext(step int) *progressbar.ProgressBar {
	if bar == nil {
		bar = progressbar.Default(100)
	}
	bar.Add(step)
	time.Sleep(40 * time.Millisecond)

	return bar
}
