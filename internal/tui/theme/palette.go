package theme

import (
	"image/color"
	"math"
	"strconv"

	"github.com/charmbracelet/lipgloss"
)

type Palette struct {
	ID        string
	Name      string
	IsLight   bool
	IsSystem  bool
	IsCustom  bool
	BG        lipgloss.Color
	Surface   lipgloss.Color
	Text      lipgloss.Color
	Muted     lipgloss.Color
	Accent    lipgloss.Color
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color
	Border    lipgloss.Color
	Selection lipgloss.Color
}

func RelativeLuminance(c color.RGBA) float64 {
	transform := func(val uint8) float64 {
		v := float64(val) / 255.0
		if v <= 0.04045 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}

	r := transform(c.R)
	g := transform(c.G)
	b := transform(c.B)
	return 0.2126*r + 0.7152*g + 0.0722*b
}

func ContrastRatio(c1Hex, c2Hex string) float64 {
	parseHex := func(h string) color.RGBA {
		if len(h) > 0 && h[0] == '#' {
			h = h[1:]
		}
		if len(h) != 6 {
			return color.RGBA{0, 0, 0, 255}
		}
		r, _ := strconv.ParseUint(h[0:2], 16, 8)
		g, _ := strconv.ParseUint(h[2:4], 16, 8)
		b, _ := strconv.ParseUint(h[4:6], 16, 8)
		return color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
	}

	l1 := RelativeLuminance(parseHex(c1Hex))
	l2 := RelativeLuminance(parseHex(c2Hex))

	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}
