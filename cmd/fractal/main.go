package main

import (
	"fmt"
	"math"
	"time"

	"github.com/andrey20005/tui_canvas"
)

// Ротация по оси Y (аналог ry в GLSL)
func ry(p [3]float64, a float64) [3]float64 {
	c, s := math.Cos(a), math.Sin(a)
	return [3]float64{
		c*p[0] + s*p[2],
		p[1],
		-s*p[0] + c*p[2],
	}
}

// Оценка расстояния до фрактала Mandelbulb (аналог mb в GLSL)
func mb(p [3]float64) [3]float64 {
	// Перестановка осей p.xzy
	z := [3]float64{p[0], p[2], p[1]}
	power := 8.0
	dr := 1.0
	t0 := 1.0

	for i := 0; i < 7; i++ {
		r := math.Sqrt(z[0]*z[0] + z[1]*z[1] + z[2]*z[2])
		if r > 2.0 {
			continue
		}

		// Вычисляем углы сферических координат
		theta := math.Atan2(z[1], z[0])
		phi := math.Asin(z[2] / r)

		dr = math.Pow(r, power-1.0)*dr*power + 1.0

		rPower := math.Pow(r, power)
		theta = theta * power
		phi = phi * power

		cosTheta, sinTheta := math.Cos(theta), math.Sin(theta)
		cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)

		z[0] = rPower*cosTheta*cosPhi + p[0]
		z[1] = rPower*sinTheta*cosPhi + p[1]
		z[2] = rPower*sinPhi + p[2]

		t0 = math.Min(t0, r)
	}

	r := math.Sqrt(z[0]*z[0] + z[1]*z[1] + z[2]*z[2])
	dist := 0.5 * math.Log(r) * r / dr
	return [3]float64{dist, t0, 0.0}
}

// Главная функция сцены (аналог f в GLSL)
func scene(p [3]float64, iTime float64) [3]float64 {
	rotatedP := ry(p, iTime*0.2)
	return mb(rotatedP)
}

// Расчет мягких теней (аналог softshadow в GLSL)
func softshadow(ro, rd [3]float64, k, iTime float64) float64 {
	akuma := 1.0
	t := 0.01
	for i := 0; i < 30; i++ { // Сокращено до 30 итераций для скорости в TUI
		pos := [3]float64{ro[0] + rd[0]*t, ro[1] + rd[1]*t, ro[2] + rd[2]*t}
		h := scene(pos, iTime)[0]
		if h < 0.001 {
			return 0.02
		}
		akuma = math.Min(akuma, k*h/t)
		t += math.Max(0.01, math.Min(h, 2.0))
	}
	return akuma
}

// Расчет нормалей методом конечных разностей (аналог nor в GLSL)
func estimateNormal(pos [3]float64, iTime float64) [3]float64 {
	eps := 0.002
	
	pX1 := scene([3]float64{pos[0] + eps, pos[1], pos[2]}, iTime)[0]
	pX2 := scene([3]float64{pos[0] - eps, pos[1], pos[2]}, iTime)[0]
	
	pY1 := scene([3]float64{pos[0], pos[1] + eps, pos[2]}, iTime)[0]
	pY2 := scene([3]float64{pos[0], pos[1] - eps, pos[2]}, iTime)[0]
	
	pZ1 := scene([3]float64{pos[0], pos[1], pos[2] + eps}, iTime)[0]
	pZ2 := scene([3]float64{pos[0], pos[1], pos[2] - eps}, iTime)[0]

	nX := pX1 - pX2
	nY := pY1 - pY2
	nZ := pZ1 - pZ2

	lenN := math.Sqrt(nX*nX + nY*nY + nZ*nZ)
	if lenN == 0 {
		return [3]float64{0, 1, 0}
	}
	return [3]float64{nX / lenN, nY / lenN, nZ / lenN}
}

// Функция марчинга лучей (аналог intersect в GLSL)
func intersect(ro, rd [3]float64, iTime float64) [3]float64 {
	t := 1.0
	resT := -1.0
	resY := 0.0
	maxError := 1000.0
	os := 0.0
	pd := 100.0
	step := 0.0

	// Ограничено до 36 шагов (вместо 48), чтобы процессор не плавился в TUI
	for i := 0; i < 36; i++ {
		if t > 8.0 {
			break
		}

		pos := [3]float64{ro[0] + rd[0]*t, ro[1] + rd[1]*t, ro[2] + rd[2]*t}
		c := scene(pos, iTime)
		d := c[0]

		if d > os {
			os = 0.4 * d * d / pd
			step = d + os
			pd = d
		} else {
			step = -os
			os = 0.0
			pd = 100.0
			d = 1.0
		}

		errorVal := d / t
		if errorVal < maxError {
			maxError = errorVal
			resT = t
			resY = c[1]
		}
		t += step
	}

	if t > 8.0 {
		resT = -1.0
	}
	return [3]float64{resT, resY, 0.0}
}

func RenderMandelbulb(x, y, iTime float64) tuicanvas.Color {
	// Твой движок дает x и y в диапазоне [-1, 1], где центр в (0,0)
	// Эмулируем нормализованные координаты q от 0 до 1 для виньетирования
	qX := x*0.5 + 0.5
	qY := y*0.5 + 0.5

	// Настройка камеры (на основе тригонометрии времени из оригинала)
	stime := 0.7 + 0.3*math.Sin(iTime*0.4)
	ctime := 0.7 + 0.3*math.Cos(iTime*0.4)

	ro := [3]float64{0.0, 4.5 * stime * ctime, 4.5 * (1.0 - stime*ctime)}
	ta := [3]float64{0.0, 0.0, 0.0}

	// Считаем базисный вектор cf (направление камеры)
	cfX, cfY, cfZ := ta[0]-ro[0], ta[1]-ro[1], ta[2]-ro[2]
	lenCf := math.Sqrt(cfX*cfX + cfY*cfY + cfZ*cfZ)
	cfX, cfY, cfZ = cfX/lenCf, cfY/lenCf, cfZ/lenCf

	// Вектор cs (cross product направления и мирового верха)
	csX := cfZ * 1.0 // cross(cf, [0,1,0])
	csZ := -cfX * 1.0
	lenCs := math.Sqrt(csX*csX + csZ*csZ)
	if lenCs > 0 {
		csX, csZ = csX/lenCs, csZ/lenCs
	}

	// Вектор cu (cross product бока и направления)
	cuX := csZ*cfY
	cuY := csX*cfZ - csZ*cfX
	cuZ := -csX * cfY

	// Формируем финальный луч rd (transform from view to world)
	rdX := x*csX + y*cuX + 2.0*cfX
	rdY := y*cuY + 2.0*cfY
	rdZ := x*csZ + y*cuZ + 2.0*cfZ
	lenRd := math.Sqrt(rdX*rdX + rdY*rdY + rdZ*rdZ)
	rd := [3]float64{rdX / lenRd, rdY / lenRd, rdZ / lenRd}

	// Окружение: свет, небо и бэкграунд
	sundir := [3]float64{0.1, 0.8, 0.6}
	
	bgR := math.Exp(y-2.0) * 0.4
	bgG := math.Exp(y-2.0) * 1.6
	bgB := math.Exp(y-2.0) * 1.0

	halo := math.Max(0.0, math.Min(1.0, (-ro[0]*rd[0] - ro[1]*rd[1] - ro[2]*rd[2])/math.Sqrt(ro[0]*ro[0]+ro[1]*ro[1]+ro[2]*ro[2])))
	haloPower := math.Pow(halo, 17.0)

	colR := bgR + 1.0*haloPower
	colG := bgG + 0.8*haloPower
	colB := bgB + 0.4*haloPower

	// Трассировка
	res := intersect(ro, rd, iTime)
	if res[0] > 0.0 {
		tHit := res[0]
		pHit := [3]float64{ro[0] + rd[0]*tHit, ro[1] + rd[1]*tHit, ro[2] + rd[2]*tHit}
		n := estimateNormal(pHit, iTime)
		shadow := softshadow(pHit, sundir, 10.0, iTime)

		dif := math.Max(0.0, n[0]*sundir[0]+n[1]*sundir[1]+n[2]*sundir[2])
		sky := 0.6 + 0.4*math.Max(0.0, n[1])
		bac := math.Max(0.0, 0.3+0.7*(-sundir[0]*n[0]-1.0*n[1]-sundir[2]*n[2]))
		
		// Рефлект вектора rd относительно нормали n
		dotRDN := rd[0]*n[0] + rd[1]*n[1] + rd[2]*n[2]
		refX := rd[0] - 2.0*dotRDN*n[0]
		refY := rd[1] - 2.0*dotRDN*n[1]
		refZ := rd[2] - 2.0*dotRDN*n[2]
		dotSunRef := sundir[0]*refX + sundir[1]*refY + sundir[2]*refZ
		spe := math.Max(0.0, math.Pow(math.Max(0.0, math.Min(1.0, dotSunRef)), 10.0))

		linR := 4.5*1.64*dif*shadow + 0.8*bac*1.64 + 0.6*sky*0.6*shadow + 3.0*spe*shadow
		linG := 4.5*1.27*dif*shadow + 0.8*bac*1.27 + 0.6*sky*1.5*shadow + 3.0*spe*shadow
		linB := 4.5*0.99*dif*shadow + 0.8*bac*0.99 + 0.6*sky*1.0*shadow + 3.0*spe*shadow

		resY := math.Pow(math.Max(0.0, math.Min(1.0, res[1])), 0.55)
		tc0R := 0.5 + 0.5*math.Sin(3.0+4.2*resY+0.0)
		tc0G := 0.5 + 0.5*math.Sin(3.0+4.2*resY+0.5)
		tc0B := 0.5 + 0.5*math.Sin(3.0+4.2*resY+1.0)

		colR = linR * 0.9 * 0.2 * tc0R
		colG = linG * 0.8 * 0.2 * tc0G
		colB = linB * 0.6 * 0.2 * tc0B

		// Туман (смешивание с бэкграундом вдали)
		fogFactor := 1.0 - math.Exp(-0.001*tHit*tHit)
		colR = colR*(1.0-fogFactor) + bgR*fogFactor
		colG = colG*(1.0-fogFactor) + bgG*fogFactor
		colB = colB*(1.0-fogFactor) + bgB*fogFactor
	}

	// Пост-обработка: гамма-коррекция, контраст, виньетирование
	colR = math.Pow(math.Max(0.0, math.Min(1.0, colR)), 0.45)
	colG = math.Pow(math.Max(0.0, math.Min(1.0, colG)), 0.45)
	colB = math.Pow(math.Max(0.0, math.Min(1.0, colB)), 0.45)

	colR = colR*0.6 + 0.4*colR*colR*(3.0-2.0*colR)
	colG = colG*0.6 + 0.4*colG*colG*(3.0-2.0*colG)
	colB = colB*0.6 + 0.4*colB*colB*(3.0-2.0*colB)

	gray := (colR + colG + colB) * 0.333
	colR = colR*(1.0 - (-0.5)) + gray*(-0.5)
	colG = colG*(1.0 - (-0.5)) + gray*(-0.5)
	colB = colB*(1.0 - (-0.5)) + gray*(-0.5)

	vignette := 0.5 + 0.5*math.Pow(16.0*qX*qY*(1.0-qX)*(1.0-qY), 0.7)
	colR *= vignette
	colG *= vignette
	colB *= vignette

	return tuicanvas.NewColorFloat(colR, colG, colB)
}

func main() {
	screen, err := tuicanvas.NewScreen("debug.log")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}
	defer screen.Close()

	// 3D Рэймэйчинг тяжелый, поэтому ставим лимит тикера поспокойнее (например, ~30 кадров)
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	startTime := time.Now()
	lastTime := time.Now()
	textShader := tuicanvas.AutoContrastShader{}

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			fps := 1.0 / now.Sub(lastTime).Seconds()
			lastTime = now

			iTime := time.Since(startTime).Seconds()

			screen.Draw(func(canvas *tuicanvas.Canvas, text *tuicanvas.TextLayer) {
				// Отрендерить 3D Мандельбульб фрактал!
				canvas.FillShaderCoords(func(x, y float64) tuicanvas.Color {
					return RenderMandelbulb(x, y, iTime)
				})

				text.Clear()
				text.PrintAt(1, int(text.Height())-2, fmt.Sprintf("MANDELBULB 3D | FPS: %.1f", fps), textShader)
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
