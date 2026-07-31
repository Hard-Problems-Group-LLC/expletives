package terminal

type ansiRGB struct {
	red   uint8
	green uint8
	blue  uint8
}

// The palette is the conventional 16-color xterm baseline. Actual terminal
// themes may choose different visible values; the nearest-index decision
// remains deterministic for an identical logical color.
var ansi16Palette = [...]ansiRGB{
	{red: 0x00, green: 0x00, blue: 0x00},
	{red: 0x80, green: 0x00, blue: 0x00},
	{red: 0x00, green: 0x80, blue: 0x00},
	{red: 0x80, green: 0x80, blue: 0x00},
	{red: 0x00, green: 0x00, blue: 0x80},
	{red: 0x80, green: 0x00, blue: 0x80},
	{red: 0x00, green: 0x80, blue: 0x80},
	{red: 0xc0, green: 0xc0, blue: 0xc0},
	{red: 0x80, green: 0x80, blue: 0x80},
	{red: 0xff, green: 0x00, blue: 0x00},
	{red: 0x00, green: 0xff, blue: 0x00},
	{red: 0xff, green: 0xff, blue: 0x00},
	{red: 0x00, green: 0x00, blue: 0xff},
	{red: 0xff, green: 0x00, blue: 0xff},
	{red: 0x00, green: 0xff, blue: 0xff},
	{red: 0xff, green: 0xff, blue: 0xff},
}

func nearestANSI16(red, green, blue uint8) int {
	bestIndex := 0
	bestDistance := colorDistance(ansi16Palette[0], red, green, blue)
	for index := 1; index < len(ansi16Palette); index++ {
		distance := colorDistance(ansi16Palette[index], red, green, blue)
		if distance < bestDistance {
			bestDistance = distance
			bestIndex = index
		}
	}
	return bestIndex
}

func colorDistance(candidate ansiRGB, red, green, blue uint8) uint32 {
	redDelta := int32(candidate.red) - int32(red)
	greenDelta := int32(candidate.green) - int32(green)
	blueDelta := int32(candidate.blue) - int32(blue)
	return uint32(
		redDelta*redDelta +
			greenDelta*greenDelta +
			blueDelta*blueDelta,
	)
}

func foregroundCode(index int) int {
	if index < 8 {
		return 30 + index
	}
	return 90 + index - 8
}

func backgroundCode(index int) int {
	if index < 8 {
		return 40 + index
	}
	return 100 + index - 8
}
