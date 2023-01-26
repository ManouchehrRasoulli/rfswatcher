package logger

import (
	"fmt"
	"log"
)

type ColorLogger struct {
	*log.Logger
}

type Color string

const (
	ColorBlack  Color = "\u001b[30m"
	ColorRed          = "\u001b[31m"
	ColorGreen        = "\u001b[32m"
	ColorYellow       = "\u001b[33m"
	ColorBlue         = "\u001b[34m"
	ColorReset        = "\u001b[0m"
)

func NewColorLogger(lg *log.Logger) *ColorLogger {
	c := ColorLogger{
		lg,
	}
	return &c
}

func (c *ColorLogger) Printcf(color Color, format string, args ...interface{}) {
	c.Print(string(color) + fmt.Sprintf(format, args...) + string(ColorBlack))
}

func (c *ColorLogger) Printc(color Color, s string) {
	c.Print(string(color) + s + string(ColorBlack))
}
