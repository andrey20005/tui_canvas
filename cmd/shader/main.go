package main

import (
	"fmt"
	"math"
	"time"

	"github.com/andrey20005/tui_canvas"
)

// MainImageShader принимает чистые вещественные координаты x и y от холста.
// t — это время iTime в секундах.
func MainImageShader(x, y, t float64) tui_canvas.Color {
	var c [3]float64
	z := t

	l := math.Sqrt(x*x + y*y)
	if l == 0 {
		l = 0.001
	}

	uvX := x*0.5 + 0.5
	uvY := y*0.5 + 0.5

	for i := 0; i < 3; i++ {
		z += 0.07
		
		factor := (math.Sin(z) + 1.0) * math.Abs(math.Sin(l*9.0-z-z))
		nextUvX := uvX + (x/l)*factor
		nextUvY := uvY + (y/l)*factor

		modX := math.Mod(nextUvX, 1.0)
		if modX < 0 { modX += 1.0 }
		modY := math.Mod(nextUvY, 1.0)
		if modY < 0 { modY += 1.0 }

		lenVec := math.Sqrt((modX-0.5)*(modX-0.5) + (modY-0.5)*(modY-0.5))
		if lenVec == 0 {
			lenVec = 0.001
		}
		c[i] = 0.01 / lenVec
	}

	return tui_canvas.NewColorFloat(c[0]/l, c[1]/l, c[2]/l)
}


func main() {
	screen, err := tui_canvas.NewScreen("debug.log")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}
	defer screen.Close()

	ticker := time.NewTicker(time.Second / 60)
	defer ticker.Stop()

	startTime := time.Now()
	lastTime := time.Now()

	uiShader := tui_canvas.AutoContrastShader{}

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			frameTime := now.Sub(lastTime)
			lastTime = now
			var fps float64
			if frameTime.Seconds() > 0 {
				fps = 1.0 / frameTime.Seconds()
			}
			iTime := time.Since(startTime).Seconds()

			screen.Draw(func(canvas *tui_canvas.Canvas, text *tui_canvas.TextLayer) {
				text.Clear()
				topRow := int(text.Height()) - 2
				text.PrintAt(1, topRow, fmt.Sprintf("FPS: %.1f", fps), uiShader)

				canvas.FillShaderCoords(func(x, y float64) tui_canvas.Color {
					return MainImageShader(x, y, iTime)
				})
			})

		case keyEv := <-screen.KeyEvents():
			if keyEv.Key == "escape" || keyEv.Key == "q" || keyEv.Key == "ctrl+c" {
				return
			}
		case <-screen.MouseEvents():
		case <-screen.ResizeEvents():
		}
	}
}
