package tuicanvas

import "math"

// ==========================================
// ЛИНИИ
// ==========================================

// DrawLine рисует линию от (x0, y0) до (x1, y1) используя алгоритм Брезенхэма.
// Алгоритм работает только с целыми числами, что делает его очень быстрым.
func (c *Canvas) DrawLine(x0, y0, x1, y1 int, color Color) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	
	err := dx - dy

	for {
		c.Set(x0, y0, color)
		if x0 == x1 && y0 == y1 {
			break
		}
		
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// ==========================================
// ПРЯМОУГОЛЬНИКИ
// ==========================================

// DrawRect рисует контур прямоугольника.
func (c *Canvas) DrawRect(x, y, w, h int, color Color) {
	if w <= 0 || h <= 0 {
		return
	}
	// Рисуем 4 стороны
	c.DrawLine(x, y, x+w-1, y, color)             // Верх
	c.DrawLine(x, y+h-1, x+w-1, y+h-1, color)    // Низ
	c.DrawLine(x, y, x, y+h-1, color)             // Лево
	c.DrawLine(x+w-1, y, x+w-1, y+h-1, color)    // Право
}

// FillRect рисует закрашенный прямоугольник.
// Оптимизирован: прямой доступ к слайсам без лишних вызовов Set.
func (c *Canvas) FillRect(x, y, w, h int, color Color) {
	if w <= 0 || h <= 0 {
		return
	}

	x0, y0 := x, y
	x1, y1 := x+w, y+h

	// Обрезаем (clipping) координаты, чтобы не выйти за пределы массива
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > int(c.width) {
		x1 = int(c.width)
	}
	if y1 > int(c.height) {
		y1 = int(c.height)
	}

	// Прямая запись в память — самый быстрый способ
	for iy := y0; iy < y1; iy++ {
		row := c.data[iy]
		for ix := x0; ix < x1; ix++ {
			row[ix] = color
		}
	}
}

// ==========================================
// ОКРУЖНОСТИ
// ==========================================

// DrawCircle рисует контур окружности используя алгоритм средней точки (Midpoint).
func (c *Canvas) DrawCircle(cx, cy, r int, color Color) {
	if r < 0 {
		return
	}
	
	x := 0
	y := r
	d := 1 - r

	for x <= y {
		c.drawCirclePoints(cx, cy, x, y, color)
		if d <= 0 {
			d += 2*x + 3
		} else {
			d += 2*(x-y) + 5
			y--
		}
		x++
	}
}

// drawCirclePoints вспомогательная функция для симметричной отрисовки 8 точек окружности.
func (c *Canvas) drawCirclePoints(cx, cy, x, y int, color Color) {
	c.Set(cx+x, cy+y, color)
	c.Set(cx-x, cy+y, color)
	c.Set(cx+x, cy-y, color)
	c.Set(cx-x, cy-y, color)
	c.Set(cx+y, cy+x, color)
	c.Set(cx-y, cy+x, color)
	c.Set(cx+y, cy-x, color)
	c.Set(cx-y, cy-x, color)
}

// FillCircle рисует закрашенную окружность.
// Использует math.Sqrt для вычисления ширины линии на каждой высоте.
// Для размеров терминала это работает мгновенно и избавляет от сложных алгоритмов заполнения.
func (c *Canvas) FillCircle(cx, cy, r int, color Color) {
	if r < 0 {
		return
	}

	x := 0
	y := r
	d := 1 - r

	for x <= y {
		// Рисуем 4 горизонтальные линии между симметричными точками
		c.drawHLine(cx-x, cx+x, cy+y, color)
		c.drawHLine(cx-x, cx+x, cy-y, color)
		c.drawHLine(cx-y, cx+y, cy+x, color)
		c.drawHLine(cx-y, cx+y, cy-x, color)

		if d <= 0 {
			d += 2*x + 3
		} else {
			d += 2*(x-y) + 5
			y--
		}
		x++
	}
}

// drawHLine рисует горизонтальную линию от x1 до x2 на строке y.
// Корректно обрабатывает выход за пределы холста.
func (c *Canvas) drawHLine(x1, x2, y int, color Color) {
	if y < 0 || uint(y) >= c.height {
		return
	}
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if x1 < 0 {
		x1 = 0
	}
	if uint(x2) >= c.width {
		x2 = int(c.width) - 1
	}
	row := c.data[y]
	for ix := x1; ix <= x2; ix++ {
		row[ix] = color
	}
}

// ==========================================
// ЭЛЛИПСЫ
// ==========================================

// DrawEllipseInRect рисует контур эллипса, вписанного в прямоугольник.
// (x, y) — координаты левого нижнего угла прямоугольника (ось Y направлена вверх).
// w, h — ширина и высота прямоугольника.
// Эллипс касается всех четырёх сторон прямоугольника, но не выходит за его пределы.
func (c *Canvas) DrawEllipse(x, y, w, h int, color Color) {
	if w <= 0 || h <= 0 {
		return
	}

	// Преобразуем координаты: x0, y0 - левый верхний угол, x1, y1 - правый нижний
	// (так как ось Y направлена вверх, y - это нижний край)
	x0 := x
	y1 := y
	x1 := x + w - 1
	y0 := y + h - 1

	a := abs(x1 - x0)
	b := abs(y1 - y0)
	b1 := b & 1

	dx := 4*(1-a)*b*b
	dy := 4*(b1+1)*a*a
	err := dx + dy + b1*a*a

	if x0 > x1 {
		x0 = x1
		x1 += a
	}
	if y0 > y1 {
		y0 = y1
	}
	y0 += (b + 1) / 2
	y1 = y0 - b1

	a *= 8 * a
	b1 = 8 * b * b

	for x0 <= x1 {
		c.Set(x1, y0, color)
		c.Set(x0, y0, color)
		c.Set(x0, y1, color)
		c.Set(x1, y1, color)

		e2 := 2 * err
		if e2 <= dy {
			y0++
			y1--
			dy += a
			err += dy
		}
		if e2 >= dx || 2*err > dy {
			x0++
			x1--
			dx += b1
			err += dx
		}
	}

	for y0-y1 < b {
		c.Set(x0-1, y0, color)
		c.Set(x1+1, y0, color)
		y0++
		c.Set(x0-1, y1, color)
		c.Set(x1+1, y1, color)
		y1--
	}
}

// FillEllipseInRect рисует эллипс, вписанный в прямоугольник.
// (x, y) — координаты левого нижнего угла прямоугольника (ось Y направлена вверх).
// w, h — ширина и высота прямоугольника.
// Эллипс касается всех четырёх сторон прямоугольника, но не выходит за его пределы.
func (c *Canvas) FillEllipse(x, y, w, h int, color Color) {
	if w <= 0 || h <= 0 {
		return
	}

	x0 := x
	y1 := y
	x1 := x + w - 1
	y0 := y + h - 1

	a := abs(x1 - x0)
	b := abs(y1 - y0)
	b1 := b & 1

	dx := 4*(1-a)*b*b
	dy := 4*(b1+1)*a*a
	err := dx + dy + b1*a*a

	if x0 > x1 {
		x0 = x1
		x1 += a
	}
	if y0 > y1 {
		y0 = y1
	}
	y0 += (b + 1) / 2
	y1 = y0 - b1

	a *= 8 * a
	b1 = 8 * b * b

	for x0 <= x1 {
		c.drawHLine(x0, x1, y0, color)
		c.drawHLine(x0, x1, y1, color)

		e2 := 2 * err
		if e2 <= dy {
			y0++
			y1--
			dy += a
			err += dy
		}
		if e2 >= dx || 2*err > dy {
			x0++
			x1--
			dx += b1
			err += dx
		}
	}

	for y0-y1 < b {
		c.drawHLine(x0-1, x1+1, y0, color)
		y0++
		c.drawHLine(x0-1, x1+1, y1, color)
		y1--
	}
}
	

// ==========================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ==========================================

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ==========================================
// ФИГУРЫ С ШЕЙДЕРАМИ
// ==========================================

// FigureShaderFn - функция шейдера для фигур.
// Принимает:
//   - absX, absY: нормализованные координаты на холсте (от -1.0 до 1.0)
//   - relX, relY: координаты относительно центра фигуры (в пикселях)
//   - idxX, idxY: абсолютные индексы пикселя
// Возвращает цвет и альфа-канал (0.0 - прозрачный, 1.0 - непрозрачный)
type FigureShaderFn func(absX, absY float64, relX, relY float64, idxX, idxY uint) (Color, float64)

// FillRectShader рисует закрашенный прямоугольник с применением шейдера.
func (c *Canvas) FillRectShader(x, y, w, h int, shader FigureShaderFn) {
	if w <= 0 || h <= 0 {
		return
	}

	x0, y0 := x, y
	x1, y1 := x+w, y+h

	// Обрезаем координаты
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > int(c.width) {
		x1 = int(c.width)
	}
	if y1 > int(c.height) {
		y1 = int(c.height)
	}

	// Центр фигуры для относительных координат
	centerX := float64(x + w/2)
	centerY := float64(y + h/2)

	// Предварительно вычисляем коэффициенты для нормализованных координат
	canvasW := float64(c.width)
	canvasH := float64(c.height)
	var k float64
	if canvasW > canvasH {
		k = 2.0 / canvasH
	} else {
		k = 2.0 / canvasW
	}
	bx := -((canvasW - 1.0) / 2.0) * k
	by := -((canvasH - 1.0) / 2.0) * k

	for iy := y0; iy < y1; iy++ {
		row := c.data[iy]
		fy := float64(iy)*k + by
		relY := float64(iy) - centerY

		for ix := x0; ix < x1; ix++ {
			fx := float64(ix)*k + bx
			relX := float64(ix) - centerX

			color, alpha := shader(fx, fy, relX, relY, uint(ix), uint(iy))
			row[ix] = row[ix].Mix(color, alpha)
		}
	}
}

// FillCircleShader рисует закрашенную окружность с применением шейдера.
func (c *Canvas) FillCircleShader(cx, cy, r int, shader FigureShaderFn) {
	if r < 0 {
		return
	}
	r2 := r * r
	// Предварительно вычисляем коэффициенты для нормализованных координат
	canvasW := float64(c.width)
	canvasH := float64(c.height)
	var k float64
	if canvasW > canvasH {
		k = 2.0 / canvasH
	} else {
		k = 2.0 / canvasW
	}
	bx := -((canvasW - 1.0) / 2.0) * k
	by := -((canvasH - 1.0) / 2.0) * k
	for y := -r; y <= r; y++ {
		halfWidth := int(math.Sqrt(float64(r2 - y*y)))
		yIdx := cy + y
		if yIdx < 0 || uint(yIdx) >= c.height {
			continue
		}
		row := c.data[yIdx]
		xStart := cx - halfWidth
		xEnd := cx + halfWidth
		if xStart < 0 { xStart = 0 }
		if xEnd >= int(c.width) { xEnd = int(c.width) - 1 }

		fy := float64(yIdx)*k + by
		relY := float64(y)

		for ix := xStart; ix <= xEnd; ix++ {
			fx := float64(ix)*k + bx
			relX := float64(ix - cx)

			color, alpha := shader(fx, fy, relX, relY, uint(ix), uint(yIdx))
			row[ix] = row[ix].Mix(color, alpha)
		}
	}
}

// DrawLineShader рисует линию с применением шейдера.
// Полезно для создания градиентных линий или анимированных эффектов.
func (c *Canvas) DrawLineShader(x0, y0, x1, y1 int, shader FigureShaderFn) {
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)

	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}

	err := dx - dy

	// Центр линии для относительных координат
	centerX := float64(x0+x1) / 2.0
	centerY := float64(y0+y1) / 2.0

	// Предварительно вычисляем коэффициенты для нормализованных координат
	canvasW := float64(c.width)
	canvasH := float64(c.height)
	var k float64
	if canvasW > canvasH {
		k = 2.0 / canvasH
	} else {
		k = 2.0 / canvasW
	}
	bx := -((canvasW - 1.0) / 2.0) * k
	by := -((canvasH - 1.0) / 2.0) * k

	for {
		fx := float64(x0)*k + bx
		fy := float64(y0)*k + by
		relX := float64(x0) - centerX
		relY := float64(y0) - centerY

		if x0 >= 0 && y0 >= 0 && uint(x0) < c.width && uint(y0) < c.height {
			color, alpha := shader(fx, fy, relX, relY, uint(x0), uint(y0))
			c.data[y0][x0] = c.data[y0][x0].Mix(color, alpha)
		}

		if x0 == x1 && y0 == y1 {
			break
		}

		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

