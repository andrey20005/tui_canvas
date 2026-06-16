package tuicanvas

import (
	"sync"
)

// Canvas представляет собой двумерный холст для рисования в терминале.
type Canvas struct {
	data   [][]Color
	width  uint
	height uint
	mu     sync.Mutex
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

// Resize изменяет размер холста. Старое изображение выравнивается по центру.
// Если новый размер больше — свободное пространство заполняется черным цветом.
// Если новый размер меньше — изображение центрированно обрезается.
func (c *Canvas) Resize(newWidth, newHeight uint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Если размеры не изменились, ничего не делаем
	if c.width == newWidth && c.height == newHeight {
		return
	}

	// Создаем новую матрицу и заполняем её черным цветом
	newData := make([][]Color, newHeight)
	for y := uint(0); y < newHeight; y++ {
		newData[y] = make([]Color, newWidth)
		for x := uint(0); x < newWidth; x++ {
			newData[y][x] = ColorBlack
		}
	}

	// Вычисляем начальные точки для копирования (центрирование)
	// Формула: (ШиринаНовая - ШиринаСтарая) / 2
	// Использован int, так как значения могут быть отрицательными (при уменьшении холста)
	offsetX := (int(newWidth) - int(c.width)) / 2
	offsetY := (int(newHeight) - int(c.height)) / 2

	// Определяем границы пересечения старого и нового холста
	// Чтобы не выйти за пределы массивов при обрезке или расширении
	oldStartY := 0
	if offsetY < 0 {
		oldStartY = -offsetY
	}
	newStartY := 0
	if offsetY > 0 {
		newStartY = offsetY
	}

	oldStartX := 0
	if offsetX < 0 {
		oldStartX = -offsetX
	}
	newStartX := 0
	if offsetX > 0 {
		newStartX = offsetX
	}

	// Копируем пиксели из старой матрицы в новую с учетом смещения
	for y := 0; ; y++ {
		oldY := oldStartY + y
		newY := newStartY + y

		// Прерываемся, если вышли за границы любого из холстов
		if oldY >= int(c.height) || newY >= int(newHeight) {
			break
		}

		for x := 0; ; x++ {
			oldX := oldStartX + x
			newX := newStartX + x

			if oldX >= int(c.width) || newX >= int(newWidth) {
				break
			}

			newData[newY][newX] = c.data[oldY][oldX]
		}
	}

	// Обновляем состояние холста
	c.data = newData
	c.width = newWidth
	c.height = newHeight
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
	c.mu.Lock()
	defer c.mu.Unlock()

	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = color
		}
	}
}

// шейдеры

// Кастомные типы функций для удобства сигнатур
type ShaderFn func(x, y int) Color
type ShaderAlphaFn func(x, y int) (Color, float64)
type ShaderCoordsFn func(x, y float64) Color
type ShaderCoordsAlphaFn func(x, y float64) (Color, float64)

// FillShader индицирует каждый пиксель по его целочисленным индексам (x, y)
func (c *Canvas) FillShader(shader ShaderFn) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = shader(int(x), int(y))
		}
	}
}

// FillShaderAlpha красит холст с учетом альфа-канала (смешивает новый цвет с текущим фоном)
func (c *Canvas) FillShaderAlpha(shader ShaderAlphaFn) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.mu.Lock()
	defer c.mu.Unlock()

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
// Метод защищен мьютексом для потокобезопасного чтения.
func (c *Canvas) At(x, y uint) Color {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Защита от выхода за границы на случай микро-задержек при ресайзе
	if x >= c.width || y >= c.height {
		return ColorBlack
	}
	return c.data[y][x]
}

func (c *Canvas) CopyFrom(src *Canvas) {
	c.mu.Lock()
	src.mu.Lock()
	defer c.mu.Unlock()
	defer src.mu.Unlock()

	// Если размеры не совпадают, пересоздаем матрицу
	if c.width != src.width || c.height != src.height {
		c.width = src.width
		c.height = src.height
		c.data = make([][]Color, c.height)
		for y := uint(0); y < c.height; y++ {
			c.data[y] = make([]Color, c.width)
		}
	}

	// Копируем пиксели
	for y := uint(0); y < c.height; y++ {
		copy(c.data[y], src.data[y])
	}
}

