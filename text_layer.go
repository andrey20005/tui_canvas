package tuicanvas

type textCell struct {
	Is     bool // true — символ есть, false — прозрачное место
	Rune   rune
	Shader TextShader
}

type TextLayer struct {
	data   [][]textCell
	width  uint
	height uint
}

// NewTextLayer создает чистый текстовый слой заданного размера.
func NewTextLayer(w, h uint) *TextLayer {
	data := make([][]textCell, h)
	for y := uint(0); y < h; y++ {
		data[y] = make([]textCell, w)
	}
	return &TextLayer{data: data, width: w, height: h}
}

// Clear полностью очищает слой текста перед каждым новым кадром,
// сбрасывая флаги Is в false.
func (tl *TextLayer) Clear() {
	for y := uint(0); y < tl.height; y++ {
		for x := uint(0); x < tl.width; x++ {
			tl.data[y][x] = textCell{Is: false}
		}
	}
}

// Resize меняет размер текстовой матрицы при изменении окна терминала.
func (tl *TextLayer) Resize(newW, newH uint) {
	if tl.width == newW && tl.height == newH {
		return
	}

	// Создаем новую пустую (прозрачную) текстовую матрицу
	newData := make([][]textCell, newH)
	for y := uint(0); y < newH; y++ {
		newData[y] = make([]textCell, newW)
	}

	dx := (int(newW) - int(tl.width)) / 2
	dy := (int(newH) - int(tl.height)) / 2

	for y := uint(0); y < tl.height; y++ {
		ny := int(y) + dy
		if ny < 0 || ny >= int(newH) { continue }

		for x := uint(0); x < tl.width; x++ {
			nx := int(x) + dx
			if nx < 0 || nx >= int(newW) { continue }

			newData[ny][nx] = tl.data[y][x]
		}
	}

	tl.data, tl.width, tl.height = newData, newW, newH
}


// PrintAt записывает строку текста в указанные координаты, используя выбранный шейдер.
// Метод полностью безопасен: принимает любые координаты (даже отрицательные) и мягко отсекает лишнее.
// Символ \n не переносит строку, а превращается в пробел.
func (tl *TextLayer) PrintAt(startX, startY int, text string, shader TextShader) {
	// Если строка гарантированно мимо экрана по вертикали — выходим сразу
	if startY < 0 || startY >= int(tl.height) {
		return
	}

	// Переводим строку в срез рун для корректной работы с UTF-8 (русскими буквами)
	runes := []rune(text)

	for i, r := range runes {
		currentX := startX + i

		// Мягкое отсечение (Clipping) границ
		if currentX < 0 {
			continue // Буква левее экрана — пропускаем её
		}
		if currentX >= int(tl.width) {
			break // Буква правее экрана — остаток строки тоже не влезет, выходим
		}

		// Заменяем переносы строк на пробелы, чтобы не ломать разметку
		if r == '\n' || r == '\r' {
			r = ' '
		}

		// Записываем ячейку в матрицу
		tl.data[startY][currentX] = textCell{
			Is:     true,
			Rune:   r,
			Shader: shader,
		}
	}
}

// Width возвращает ширину текстового слоя.
func (tl *TextLayer) Width() uint { return tl.width }

// Height возвращает высоту текстового слоя.
func (tl *TextLayer) Height() uint { return tl.height }
