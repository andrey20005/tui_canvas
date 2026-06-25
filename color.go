package tui_canvas

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type RGB struct {
	channels [3]float64
}

var (
	ColorBlack   = RGB{channels: [3]float64{0.0, 0.0, 0.0}}
	ColorWhite   = RGB{channels: [3]float64{1.0, 1.0, 1.0}}
	ColorRed     = RGB{channels: [3]float64{1.0, 0.0, 0.0}}
	ColorGreen   = RGB{channels: [3]float64{0.0, 1.0, 0.0}}
	ColorBlue    = RGB{channels: [3]float64{0.0, 0.0, 1.0}}
	ColorYellow  = RGB{channels: [3]float64{1.0, 1.0, 0.0}}
	ColorCyan    = RGB{channels: [3]float64{0.0, 1.0, 1.0}}
	ColorMagenta = RGB{channels: [3]float64{1.0, 0.0, 1.0}}
	ColorGray    = RGB{channels: [3]float64{0.5, 0.5, 0.5}}
)

// NewRGB создает цвет из float64 в диапазоне [0.0, 1.0]
func NewRGB(r, g, b float64) RGB {
	return RGB{channels: [3]float64{r, g, b}}
}

// NewRGBUint создает цвет из привычных uint8 (0-255)
func NewRGBUint(r, g, b uint8) RGB {
	return RGB{channels: [3]float64{
		float64(r) / 255.0,
		float64(g) / 255.0,
		float64(b) / 255.0,
	}}
}

// NewColorHexNum создает цвет из числового HEX (например, 0xFF00FF)
func NewColorHexNum(hex uint32) RGB {
	r := uint8((hex >> 16) & 0xFF)
	g := uint8((hex >> 8) & 0xFF)
	b := uint8(hex & 0xFF)
	return NewRGBUint(r, g, b)
}

// NewColorHexString создает цвет из строки вида "#RRGGBB" или "#RGB" (регистр не важен)
// Если строка невалидна, возвращает черный цвет и ошибку
func NewColorHexString(hexStr string) (RGB, error) {
	// Убираем пробелы и символ '#'
	hexStr = strings.TrimSpace(hexStr)
	hexStr = strings.TrimPrefix(hexStr, "#")

	var rStr, gStr, bStr string

	switch len(hexStr) {
	case 3: // Короткий формат вроде "F0F" -> "FF00FF"
		rStr = string([]byte{hexStr[0], hexStr[0]})
		gStr = string([]byte{hexStr[1], hexStr[1]})
		bStr = string([]byte{hexStr[2], hexStr[2]})
	case 6: // Стандартный формат "FF00FF"
		rStr = hexStr[0:2]
		gStr = hexStr[2:4]
		bStr = hexStr[4:6]
	default:
		return RGB{}, fmt.Errorf("invalid hex color format: %s", hexStr)
	}

	r, err1 := strconv.ParseUint(rStr, 16, 8)
	g, err2 := strconv.ParseUint(gStr, 16, 8)
	b, err3 := strconv.ParseUint(bStr, 16, 8)

	if err1 != nil || err2 != nil || err3 != nil {
		return RGB{}, fmt.Errorf("invalid hex characters in color: %s", hexStr)
	}

	return NewRGBUint(uint8(r), uint8(g), uint8(b)), nil
}

func (c RGB) R() float64 { return c.channels[0] }

func (c RGB) G() float64 { return c.channels[1] }

func (c RGB) B() float64 { return c.channels[2] }

func (c RGB) Brightness() float64 {
	return c.channels[0]*0.299 + c.channels[1]*0.587 + c.channels[2]*0.114
}

// ToRGB возвращает срез []uint8 из 3 элементов (0-255)
func (c RGB) ToRGB() [3]uint8 {
	return [3]uint8{
		clampRGB(c.channels[0]),
		clampRGB(c.channels[1]),
		clampRGB(c.channels[2]),
	}
}

// ToHexNum возвращает цвет в виде числа uint32 (например, 0xFF00FF)
func (c RGB) ToHexNum() uint32 {
	rgb := c.ToRGB()
	return (uint32(rgb[0]) << 16) | (uint32(rgb[1]) << 8) | uint32(rgb[2])
}

// ToHexString возвращает цвет в виде строки формата "#RRGGBB" в верхнем регистре
func (c RGB) ToHexString() string {
	rgb := c.ToRGB()
	return fmt.Sprintf("#%02X%02X%02X", rgb[0], rgb[1], rgb[2])
}

// Mix выполняет линейную интерполяцию между текущим цветом и цветом y.
// Параметр t определяет пропорцию смешивания и обычно находится в диапазоне [0.0, 1.0].
// Формула: x * (1 - t) + y * t
func (x RGB) Mix(y RGB, t float64) RGB {
	// Ограничиваем t в пределах [0.0, 1.0] для безопасности вычислений
	if t < 0.0 {
		t = 0.0
	} else if t > 1.0 {
		t = 1.0
	}

	return RGB{channels: [3]float64{
		x.channels[0]*(1.0-t) + y.channels[0]*t,
		x.channels[1]*(1.0-t) + y.channels[1]*t,
		x.channels[2]*(1.0-t) + y.channels[2]*t,
	}}
}

func clampRGB(val float64) uint8 {
	res := math.Round(val * 255.0)
	if res < 0.0 {
		return 0
	}
	if res > 255.0 {
		return 255
	}
	return uint8(res)
}
