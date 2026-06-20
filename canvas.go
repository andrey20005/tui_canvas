package tuicanvas

// Canvas представляет собой двумерный холст для рисования в терминале.
type Canvas struct {
	data   [][]Color
	width  uint
	height uint
}

// NewCanvas создает новый холст заданного размера, заполненный черным цветом.
func NewCanvas(width, height uint) *Canvas {
	data := make([][]Color, height)
	for y := uint(0); y < height; y++ {
		data[y] = make([]Color, width)
		for x := uint(0); x < width; x++ {
			data[y][x] = ColorBlack
		}
	}
	return &Canvas{
		data:   data,
		width:  width,
		height: height,
	}
}

// resize изменяет размер холста. Старое изображение выравнивается по центру.
// Если новый размер больше — свободное пространство заполняется черным цветом.
// Если новый размер меньше — изображение центрированно обрезается.
func (c *Canvas) resize(newW, newH uint) {
	if c.width == newW && c.height == newH {
		return
	}

	// Создаем новую матрицу, заполненную черным цветом
	newData := make([][]Color, newH)
	for y := uint(0); y < newH; y++ {
		newData[y] = make([]Color, newW)
		for x := uint(0); x < newW; x++ {
			newData[y][x] = ColorBlack
		}
	}

	// Смещения для центрирования (могут быть отрицательными)
	dx := (int(newW) - int(c.width)) / 2
	dy := (int(newH) - int(c.height)) / 2

	// Копируем только пересекающиеся области
	for y := uint(0); y < c.height; y++ {
		ny := int(y) + dy
		if ny < 0 || ny >= int(newH) {
			continue
		}

		for x := uint(0); x < c.width; x++ {
			nx := int(x) + dx
			if nx < 0 || nx >= int(newW) {
				continue
			}

			newData[ny][nx] = c.data[y][x]
		}
	}

	c.data, c.width, c.height = newData, newW, newH
}

// Width возвращает текущую ширину холста
func (c *Canvas) Width() uint { return c.width }

// Height возвращает текущую высоту холста
func (c *Canvas) Height() uint { return c.height }

// ==========================================
// МЕТОДЫ ОКРАШИВАНИЯ
// ==========================================

// Fill закрашивает весь холст одним сплошным цветом.
// Это самый быстрый способ очистить экран или задать базовый фон.
func (c *Canvas) Fill(color Color) {
	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = color
		}
	}
}

// Кастомные типы функций для удобства сигнатур
type ShaderFn func(x, y int) Color
type ShaderAlphaFn func(x, y int) (Color, float64)
type ShaderCoordsFn func(x, y float64) Color
type ShaderCoordsAlphaFn func(x, y float64) (Color, float64)

// FillShader индицирует каждый пиксель по его целочисленным индексам (x, y)
func (c *Canvas) FillShader(shader ShaderFn) {
	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = shader(int(x), int(y))
		}
	}
}

// FillShaderAlpha красит холст с учетом альфа-канала (смешивает новый цвет с текущим фоном)
func (c *Canvas) FillShaderAlpha(shader ShaderAlphaFn) {
	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			color, alpha := shader(int(x), int(y))
			c.data[y][x] = c.data[y][x].Mix(color, alpha)
		}
	}
}

// FillShaderCoords красит холст, используя вещественные координаты от -1.0 до 1.0 по меньшей стороне.
// Математика полностью вынесена за пределы циклов, внутри — быстрый инкремент.
func (c *Canvas) FillShaderCoords(shader ShaderCoordsFn) {
	if c.width == 0 || c.height == 0 {
		return
	}

	w := float64(c.width)
	h := float64(c.height)

	// 1. Вычисляем шаг (s) ровно один раз
	var s float64
	if w > h {
		s = 2.0 / h
	} else {
		s = 2.0 / w
	}

	// 2. Находим стартовую точку (левый верхний угол холста при xIdx=0, yIdx=0)
	startX := -((w - 1.0) / 2.0) * s
	startY := -((h - 1.0) / 2.0) * s

	// 3. Быстрые циклы: вместо вызова функций мы просто прибавляем шаг s
	fy := startY
	for y := uint(0); y < c.height; y++ {
		fx := startX
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = shader(fx, fy)
			fx += s // Сдвигаемся вправо на один пиксель
		}
		fy += s // Сдвигаемся вниз на одну строку
	}
}

// FillShaderCoordsAlpha делает то же самое, но с учетом альфа-смешивания.
func (c *Canvas) FillShaderCoordsAlpha(shader ShaderCoordsAlphaFn) {
	if c.width == 0 || c.height == 0 {
		return
	}

	w := float64(c.width)
	h := float64(c.height)

	var s float64
	if w > h {
		s = 2.0 / h
	} else {
		s = 2.0 / w
	}

	startX := -((w - 1.0) / 2.0) * s
	startY := -((h - 1.0) / 2.0) * s

	fy := startY
	for y := uint(0); y < c.height; y++ {
		fx := startX
		for x := uint(0); x < c.width; x++ {
			color, alpha := shader(fx, fy)
			c.data[y][x] = c.data[y][x].Mix(color, alpha)
			fx += s
		}
		fy += s
	}
}

// GetCoords переводит целочисленные индексы пикселя в вещественные координаты.
// Точка (0.0, 0.0) — центр экрана. Меньшая из сторон имеет длину 2.0 (от -1.0 до 1.0).
func (c *Canvas) GetCoords(xIdx, yIdx uint) (float64, float64) {
	if c.width == 0 || c.height == 0 {
		return 0.0, 0.0
	}
	w := float64(c.width)
	h := float64(c.height)
	if w > h {
		s := 2. / h
		// Правильное центрирование длинной стороны:
		fx := (float64(xIdx) - (w-1.)/2.) * s
		// Короткая сторона (от -1 до 1):
		fy := (float64(yIdx) - (h-1.)/2.) * s
		return fx, fy
	} else {
		s := 2. / w
		// Короткая сторона (от -1 до 1):
		fx := (float64(xIdx) - (w-1.)/2.) * s
		// Правильное центрирование длинной стороны:
		fy := (float64(yIdx) - (h-1.)/2.) * s
		return fx, fy
	}
}

// At возвращает цвет пикселя по указанным индексам.
func (c *Canvas) At(x, y uint) Color {
	if x >= c.width || y >= c.height {
		return ColorBlack
	}
	return c.data[y][x]
}
