package main

import (
	"fmt"
	"time"

	"github.com/andrey20005/tui_canvas"
)

func main() {
	// Создаем экран и включаем опциональное логирование в файл "debug.log"
	// Если передать "", логирования не будет
	screen, err := tuicanvas.NewScreen("debug.log")
	if err != nil {
		fmt.Println("Ошибка инициализации экрана:", err)
		return
	}
	// Гарантируем, что терминал восстановится в любом случае
	defer screen.Close()

	// Создаем таймер на 30 FPS (~33мс на кадр)
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	// Координаты нашей рисуемой точки (для управления с WASD)
	playerX, playerY := 0.0, 0.0

	for {
		select {
		case <-ticker.C:
			// НАСТУПИЛ НОВЫЙ КАДР: РЕНДЕРИНГ
			screen.Draw(func(canvas *tuicanvas.Canvas, textLayer *tuicanvas.TextLayer) {
				// Заливаем фон красивым процедурным шейдером (синий градиент)
				canvas.FillShaderCoords(func(x, y float64) tuicanvas.Color {
					return tuicanvas.NewColorFloat(0.0, 0.0, (y+1.0)*0.3)
				})

				// Поверх шейдера рисуем кружок нашего "игрока" с помощью FillCoordsAlpha
				canvas.FillShaderCoordsAlpha(func(x, y float64) (tuicanvas.Color, float64) {
					// Считаем расстояние от текущих координат игрока
					dx := x - playerX
					dy := y - playerY
					if dx*dx+dy*dy < 0.02 { // Маленький радиус
						return tuicanvas.ColorYellow, 1.0
					}
					return tuicanvas.ColorBlack, 0.0
				})
			})

		case keyEv := <-screen.KeyEvents():
			// ОБРАБОТКА КЛАВИАТУРЫ
			switch keyEv.Key {
			case "escape", "q", "ctrl+c", "й":
				// Безопасный выход из игры
				return
			case "w", "up":
				playerY += 0.1
			case "s", "down":
				playerY -= 0.1
			case "a", "left":
				playerX -= 0.1
			case "d", "right":
				playerX += 0.1
			}

		case mouseEv := <-screen.MouseEvents():
			// ОБРАБОТКА МЫШИ
			// Если зажата левая кнопка мыши (MouseLeft), телепортируем желтый кружок туда
			if mouseEv.Button == tuicanvas.MouseLeft && mouseEv.IsDown {
				// Конвертируем индексы пикселей в координаты шейдера (-1.0..1.0)
				fx, fy := screen.Canvas().GetCoords(mouseEv.X, mouseEv.Y)
				playerX = fx
				playerY = fy
			}


		case <-screen.ResizeEvents():
			// ОБРАБОТКА РЕСАЙЗА
			// Сам экран Screen уже перестроил Canvas внутри handleResize,
			// нам здесь ничего делать не нужно, в следующем тике кадра 
			// графика автоматически отрисуется под новый размер.
		}
	}
}
