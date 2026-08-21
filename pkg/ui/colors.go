package ui

import "fmt"

const (
	Reset       = "\033[0m"
	AttrBold    = "\033[1m"
	AttrDim     = "\033[2m"
	AttrItalic  = "\033[3m"
	Dim         = "\033[2m"
	Italic      = "\033[3m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	Gray        = "\033[90m"
	PastelPink  = "\033[38;5;218m" // Soft Pastel Pink (ANSI 256)
	PastelRose  = "\033[38;5;212m" // Vibrant Pastel Rose (ANSI 256)
	PastelBlush = "\033[38;5;225m" // Light Pastel Blush (ANSI 256)
)

func Colorize(color, text string) string {
	return fmt.Sprintf("%s%s%s", color, text, Reset)
}

func Bold(text string) string           { return Colorize(AttrBold, text) }
func BoldCyan(text string) string       { return Colorize(AttrBold+Cyan, text) }
func BoldGreen(text string) string      { return Colorize(AttrBold+Green, text) }
func BoldYellow(text string) string     { return Colorize(AttrBold+Yellow, text) }
func BoldRed(text string) string        { return Colorize(AttrBold+Red, text) }
func BoldMagenta(text string) string    { return Colorize(AttrBold+Magenta, text) }
func BoldBlue(text string) string       { return Colorize(AttrBold+Blue, text) }
func BoldPastelPink(text string) string { return Colorize(AttrBold+PastelPink, text) }
func PastelPinkText(text string) string { return Colorize(PastelPink, text) }
func DimText(text string) string        { return Colorize(Dim, text) }
func GrayText(text string) string       { return Colorize(Gray, text) }
