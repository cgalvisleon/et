package color

import (
	"fmt"

	"github.com/cgalvisleon/et/stdrout"
)

/**
* color applies the specified color to the given arguments and returns the colored string
* @params color string, str string
* @return string
**/
func color(color string, str string) string {
	return fmt.Sprint(color, str, stdrout.Reset)
}

/**
* Purple returns a purple-colored string
* @params str string
* @return string
**/
func Purple(str string) string {
	return color(stdrout.Purple, str)
}

/**
* Green returns a green-colored string
* @params str string
* @return string
**/
func Green(str string) string {
	return color(stdrout.Green, str)
}

/**
* Red returns a red-colored string
* @params str string
* @return string
**/
func Red(str string) string {
	return color(stdrout.Red, str)
}

/**
* Yellow returns a yellow-colored string
* @params str string
* @return string
**/
func Yellow(str string) string {
	return color(stdrout.Yellow, str)
}

/**
* Blue returns a blue-colored string
* @params str string
* @return string
**/
func Blue(str string) string {
	return color(stdrout.Blue, str)
}

/**
* Cyan returns a cyan-colored string
* @params str string
* @return string
**/
func Cyan(str string) string {
	return color(stdrout.Cyan, str)
}

/**
* White returns a white-colored string
* @params str string
* @return string
**/
func White(str string) string {
	return color(stdrout.White, str)
}

/**
* Black returns a black-colored string
* @params str string
* @return string
**/
func Black(str string) string {
	return color(stdrout.Black, str)
}
