package theme

import (
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func IsNoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

func HasTrueColor() bool {
	colorTerm := strings.ToLower(os.Getenv("COLORTERM"))
	return colorTerm == "truecolor" || colorTerm == "24bit"
}

func Resolve(c lipgloss.Color) lipgloss.Color {
	colorString := string(c)
	if IsNoColor() {
		return lipgloss.Color("")
	}
	if !strings.HasPrefix(colorString, "#") || HasTrueColor() {
		return c
	}

	code := HexToANSI256(colorString)
	return lipgloss.Color(strconv.Itoa(code))
}

func HexToANSI256(hexStr string) int {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) != 6 {
		return 7
	}

	r, err1 := strconv.ParseUint(hexStr[0:2], 16, 8)
	g, err2 := strconv.ParseUint(hexStr[2:4], 16, 8)
	b, err3 := strconv.ParseUint(hexStr[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 7
	}

	bestIndex := 0
	bestDistance := math.MaxFloat64

	for index := 0; index < 256; index++ {
		colorR, colorG, colorB := ansi256RGB(index)
		diffR := float64(int(r) - colorR)
		diffG := float64(int(g) - colorG)
		diffB := float64(int(b) - colorB)

		distance := 0.299*diffR*diffR + 0.587*diffG*diffG + 0.114*diffB*diffB
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = index
		}
	}
	return bestIndex
}

func ansi256RGB(index int) (int, int, int) {
	if index < 16 {
		ansi16Table := [16][3]int{
			{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0},
			{0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
			{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0},
			{0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
		}
		return ansi16Table[index][0], ansi16Table[index][1], ansi16Table[index][2]
	}
	if index >= 232 {
		gray := 8 + (index-232)*10
		return gray, gray, gray
	}
	colorIndex := index - 16
	steps := []int{0, 95, 135, 175, 215, 255}
	b := steps[colorIndex%6]
	g := steps[(colorIndex/6)%6]
	r := steps[(colorIndex/36)%6]
	return r, g, b
}
