package tuicanvas

type TextShader interface {
	Process(r rune, topPixelColor, bottomPixelColor Color) (finalRune rune, fg Color, bg Color)
}

type TransparentTextShader struct { TextColor Color }

func (s TransparentTextShader) Process(r rune, top, bottom Color) (rune, Color, Color) {
	// Фоном буквы становится среднее арифметическое пикселей под ней (эффект стекла)
	bg := top.Mix(bottom, 0.5)
	return r, s.TextColor, bg
}

type AutoContrastShader struct{}

func (s AutoContrastShader) Process(r rune, top, bottom Color) (rune, Color, Color) {
	bg := top.Mix(bottom, 0.5)
	
	brightness := bg.Brightness()
	
	fg := ColorWhite // По умолчанию текст белый
	if brightness > 0.5 {
		fg = ColorBlack // Если фон слишком яркий, делаем текст черным для читаемости
	}
	
	return r, fg, bg
}

type GlassShader struct {
	TextColor Color
	BgColor   Color
	BgAlpha   float64
}

func (s GlassShader) Process(r rune, top, bottom Color) (rune, Color, Color) {
	canvasBg := top.Mix(bottom, 0.5)
	// Смешиваем кастомный цвет фона с фоном холста
	finalBg := canvasBg.Mix(s.BgColor, s.BgAlpha)
	return r, s.TextColor, finalBg
}

