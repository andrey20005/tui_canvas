package tui_canvas

type TextShader interface {
	Process(r rune, c RGB) (finalRune rune, fg RGB, bg RGB)
}

type TransparentTextShader struct{ TextColor RGB }

func (s TransparentTextShader) Process(r rune, c RGB) (rune, RGB, RGB) {
	return r, s.TextColor, c 
}

type AutoContrastShader struct{} 

func (s AutoContrastShader) Process(r rune, c RGB) (rune, RGB, RGB) {
	brightness := c.Brightness()

	fg := ColorWhite // По умолчанию текст белый
	if brightness > 0.5 {
		fg = ColorBlack // Если фон слишком яркий, делаем текст черным для читаемости
	}

	return r, fg, c 
}

type GlassShader struct {
	TextColor RGB
	BgColor   RGB
	BgAlpha   float64
}

func (s GlassShader) Process(r rune, c RGB) (rune, RGB, RGB) {
	return r, s.TextColor, c.Mix(s.BgColor, s.BgAlpha)
}
