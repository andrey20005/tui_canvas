package main

import (
	"fmt"
	"math"
	"time"

	"github.com/andrey20005/tui_canvas"
)

// MainImageShader — это порт знаменитого шейдера с ShaderToy.
// t — время в секундах с начала запуска.
// x, y — это вещественные координаты пикселя от FillCoords.
// aspect — соотношение сторон (w / h), нужно для точного воссоздания GLSL математики uv.
func MainImageShader(x, y, t, aspect float64) tuicanvas.Color {
	var c [3]float64
	z := t
	
	// Пересчитываем uv обратно в диапазон [0, 1] для текстурных эффектов GLSL,
	// учитывая соотношение сторон длинной оси.
	var uvX, uvY float64
	if aspect > 1.0 {
		uvX = (x / aspect + 1.0) / 2.0
		uvY = (y + 1.0) / 2.0
	} else {
		uvX = (x + 1.0) / 2.0
		uvY = (y * aspect + 1.0) / 2.0
	}

	// Считаем расстояние от центра (длину вектора p)
	l := math.Sqrt(x*x + y*y)
	
	// Предотвращаем деление на ноль в самом центре экрана
	if l == 0 {
		l = 0.001
	}

	for i := 0; i < 3; i++ {
		z += 0.07
		
		// uv += p/l * (sin(z)+1.) * abs(sin(l*9.-z-z))
		factor := (math.Sin(z) + 1.0) * math.Abs(math.Sin(l*9.0-z-z))
		
		nextUvX := uvX + (x/l)*factor
		nextUvY := uvY + (y/l)*factor
		
		// c[i] = .01 / length(mod(uv,1.)-.5)
		// В Go нет встроенного оператора mod для float, используем math.Mod
		modX := math.Mod(nextUvX, 1.0)
		if modX < 0 { modX += 1.0 } // В GLSL mod всегда положительный
		
		modY := math.Mod(nextUvY, 1.0)
		if modY < 0 { modY += 1.0 }

		// Считаем длину вектора (mod(uv, 1.) - 0.5)
		lenVec := math.Sqrt((modX-0.5)*(modX-0.5) + (modY-0.5)*(modY-0.5))
		if lenVec == 0 {
			lenVec = 0.001
		}

		c[i] = 0.01 / lenVec
	}

	// Результат: c / l
	return tuicanvas.NewColorFloat(c[0]/l, c[1]/l, c[2]/l)
}

func main() {
	screen, err := tuicanvas.NewScreen("debug.log")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}
	defer screen.Close()

	ticker := time.NewTicker(time.Second / 120)
	defer ticker.Stop()

	// Запоминаем время старта программы
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			canvas := screen.Canvas()
			
			// Вычисляем iTime (текущее время в секундах с плавающей точкой)
			iTime := time.Since(startTime).Seconds()
			
			// Вычисляем соотношение сторон для шейдера
			w := float64(canvas.Width())
			h := float64(canvas.Height())
			aspect := w / h
			if h == 0 {
				aspect = 1.0
			}

			// Отрисовываем наш ShaderToy порт!
			canvas.FillShaderCoords(func(x, y float64) tuicanvas.Color {
				return MainImageShader(x, y, iTime, aspect)
			})

			screen.Update()

		case keyEv := <-screen.KeyEvents():
			if keyEv.Key == "escape" || keyEv.Key == "q" || keyEv.Key == "ctrl+c" {
				return
			}
		case <-screen.MouseEvents():
		case <-screen.ResizeEvents():
		}
	}
}
