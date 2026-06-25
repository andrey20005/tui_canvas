package tui_canvas

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// Canvas представляет собой двумерный холст для рисования в терминале.
// Ось y везде направленна вверх, строчка с нулевым индексом нижняя
type Canvas struct {
	data   [][]RGB
	width  uint
	height uint
}

// NewCanvas создает новый холст заданного размера, заполненный черным цветом.
func NewCanvas(width, height uint) *Canvas {
	data := make([][]RGB, height)
	for y := uint(0); y < height; y++ {
		data[y] = make([]RGB, width)
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
	newData := make([][]RGB, newH)
	for y := uint(0); y < newH; y++ {
		newData[y] = make([]RGB, newW)
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
func (c *Canvas) Fill(color RGB) {
	for y := uint(0); y < c.height; y++ {
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = color
		}
	}
}

// Кастомные типы функций для удобства сигнатур
type ShaderFn func(x, y int) RGB
type ShaderAlphaFn func(x, y int) (RGB, float64)
type ShaderCoordsFn func(x, y float64) RGB
type ShaderCoordsAlphaFn func(x, y float64) (RGB, float64)

// FillShader индицирует каждый пиксель по его целочисленным индексам (x, y)
func (c *Canvas) FillShader(shader ShaderFn) {
	c.parallelY(func(y uint) {
		yInt := int(y)
		for x := uint(0); x < c.width; x++ {
			c.data[y][x] = shader(int(x), yInt)
		}
	})
}

// FillShaderAlpha красит холст с учетом альфа-канала (смешивает новый цвет с текущим фоном)
func (c *Canvas) FillShaderAlpha(shader ShaderAlphaFn) {
	c.parallelY(func(y uint) {
		yInt := int(y)
		for x := uint(0); x < c.width; x++ {
			color, alpha := shader(int(x), yInt)
			c.data[y][x] = c.data[y][x].Mix(color, alpha)
		}
	})
}

// FillShaderCoords красит холст, используя вещественные координаты от -1.0 до 1.0 по меньшей стороне.
// Математика полностью вынесена за пределы циклов, внутри — быстрый инкремент.
func (c *Canvas) FillShaderCoords(shader ShaderCoordsFn) {
	if c.width == 0 || c.height == 0 {
		return
	}

	w := float64(c.width)
	h := float64(c.height)

	// Коэффициент масштабирования (k)
	var k float64
	if w > h {
		k = 2.0 / h
	} else {
		k = 2.0 / w
	}

	// Коэффициенты смещения (b) для левого верхнего угла
	bx := -((w - 1.0) / 2.0) * k
	by := -((h - 1.0) / 2.0) * k

	c.parallelY(func(y uint) {
		// Вычисляем fy напрямую для конкретной строки через замыкание
		fy := float64(y)*k + by

		for x := uint(0); x < c.width; x++ {
			// Вычисляем fx напрямую для конкретного пикселя
			fx := float64(x)*k + bx
			c.data[y][x] = shader(fx, fy)
		}
	})
}

// FillShaderCoordsAlpha делает то же самое, но с учетом альфа-смешивания.
func (c *Canvas) FillShaderCoordsAlpha(shader ShaderCoordsAlphaFn) {
	if c.width == 0 || c.height == 0 {
		return
	}

	w := float64(c.width)
	h := float64(c.height)

	// Коэффициент масштабирования (k)
	var k float64
	if w > h {
		k = 2.0 / h
	} else {
		k = 2.0 / w
	}

	// Коэффициенты смещения (b) для левого верхнего угла
	bx := -((w - 1.0) / 2.0) * k
	by := -((h - 1.0) / 2.0) * k

	c.parallelY(func(y uint) {
		// Вычисляем fy напрямую для строки через замыкание коэффициентов
		fy := float64(y)*k + by

		for x := uint(0); x < c.width; x++ {
			// Вычисляем fx напрямую для пикселя
			fx := float64(x)*k + bx
			color, alpha := shader(fx, fy)
			c.data[y][x] = c.data[y][x].Mix(color, alpha)
		}
	})
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

// parallelY — высокопроизводительный оркестратор для обработки строк холста.
// Использует динамический пул воркеров и атомарное распределение строк для
// идеального балансирования нагрузки между ядрами процессора.
func (c *Canvas) parallelY(worker func(yIdx uint)) {
	if c.height == 0 {
		return
	}

	// 1. Быстрый путь для маленьких холстов или одноядерных систем.
	// Запуск горутин на холсте меньше 32 строк принесет больше вреда, чем пользы.
	numWorkers := runtime.NumCPU()
	if numWorkers <= 1 || c.height < 20 {
		for y := uint(0); y < c.height; y++ {
			worker(y)
		}
		return
	}

	// Ограничиваем число воркеров высотой холста, если ядер больше, чем строк
	if numWorkers > int(c.height) {
		numWorkers = int(c.height)
	}

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Атомарный счетчик текущей строки, которую нужно обработать
	var currentY uint64 = 0
	targetY := uint64(c.height)

	// 2. Запускаем пул воркеров
	for w := 0; w < numWorkers; w++ {
		go func() {
			defer wg.Done()

			for {
				// Берем следующую доступную строку и сдвигаем счетчик на 1 вперед
				y := atomic.AddUint64(&currentY, 1) - 1

				// Если все строки разобраны — воркер завершает работу
				if y >= targetY {
					return
				}

				// Передаем индекс строки в колбэк шейдера
				worker(uint(y))
			}
		}()
	}

	// Ждем, пока все воркеры разберут и обработают строки
	wg.Wait()
}

// At возвращает цвет пикселя по указанным индексам.
func (c *Canvas) At(x, y int) RGB {
	if x >= int(c.width) || y >= int(c.height) || x < 0 || y < 0 {
		return ColorBlack
	}
	return c.data[y][x]
}

func (c *Canvas) Set(x, y int, color RGB) {
	if x >= 0 && y >= 0 && uint(x) < c.width && uint(y) < c.height {
		c.data[y][x] = color
	}
}
