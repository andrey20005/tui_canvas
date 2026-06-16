package tuicanvas

import (
	"bufio"
	"os"
	"strconv"
)


func (s *Screen) Update() {
	s.backCanvas.CopyFrom(s.frozenCanvas)
	s.frozenCanvas.CopyFrom(s.currentCanvas)

	if s.backCanvas.Width() != s.frozenCanvas.Width() || s.backCanvas.Height() != s.frozenCanvas.Height() {
		s.simpleUpdate()
	} else {
		s.optimizedUpdate()
	}
}

// simpleUpdate делает полную перерисовку всего экрана (используется при старте и ресайзе)
func (s *Screen) simpleUpdate() {
	w := s.frozenCanvas.Width()
	h := s.frozenCanvas.Height()
	if w == 0 || h == 0 {
		return
	}

	os.Stdout.WriteString("\x1b[H")
	writer := bufio.NewWriterSize(os.Stdout, 256*1024)

	termRow := 1
	startY := int(h) - 1

	for y := startY; y >= 0; y -= 2 {
		writer.WriteString("\x1b[" + strconv.Itoa(termRow) + ";1H")
		termRow++

		for x := uint(0); x < w; x++ {
			topColor := s.frozenCanvas.At(x, uint(y)).ToRGB()
			var bottomColor [3]uint8
			if y-1 >= 0 {
				bottomColor = s.frozenCanvas.At(x, uint(y-1)).ToRGB()
			}

			writer.WriteString("\x1b[38;2;")
			writer.WriteString(strconv.Itoa(int(topColor[0])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(topColor[1])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(topColor[2])))
			writer.WriteString(";48;2;")
			writer.WriteString(strconv.Itoa(int(bottomColor[0])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(bottomColor[1])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(bottomColor[2])))
			writer.WriteString("m▀")
		}
	}
	writer.WriteString("\x1b[0m")
	writer.Flush()
}

func (s *Screen) optimizedUpdate() {
	w := s.frozenCanvas.Width()
	h := s.frozenCanvas.Height()
	if w == 0 || h == 0 {
		return
	}

	// Копим измененные байты в буфере, чтобы выплюнуть их одной порцией
	writer := bufio.NewWriterSize(os.Stdout, 256*1024)
	
	termRow := 1
	startY := int(h) - 1

	for y := startY; y >= 0; y -= 2 {
		for x := uint(0); x < w; x++ {
			
			// Получаем пиксели текущего кадра
			currentTop := s.frozenCanvas.At(x, uint(y))
			var currentBottom Color
			if y-1 >= 0 {
				currentBottom = s.frozenCanvas.At(x, uint(y-1))
			}

			// Получаем пиксели прошлого кадра
			backTop := s.backCanvas.At(x, uint(y))
			var backBottom Color
			if y-1 >= 0 {
				backBottom = s.backCanvas.At(x, uint(y-1))
			}

			// Если цвета верхнего и нижнего пикселя полностью совпадают с прошлым кадром — пропускаем!
			if currentTop == backTop && currentBottom == backBottom {
				continue
			}

			// Если нашли разницу — прыгаем курсором прямо на координаты этого символа в терминале.
			// termRow — текущая строка, x+1 — колонка (терминал считает координаты с 1)
			writer.WriteString("\x1b[" + strconv.Itoa(termRow) + ";" + strconv.Itoa(int(x+1)) + "H")

			topRGB := currentTop.ToRGB()
			bottomRGB := currentBottom.ToRGB()

			// Отрисовываем один обновленный символ ▀
			writer.WriteString("\x1b[38;2;")
			writer.WriteString(strconv.Itoa(int(topRGB[0])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(topRGB[1])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(topRGB[2])))
			writer.WriteString(";48;2;")
			writer.WriteString(strconv.Itoa(int(bottomRGB[0])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(bottomRGB[1])))
			writer.WriteString(";")
			writer.WriteString(strconv.Itoa(int(bottomRGB[2])))
			writer.WriteString("m▀")
		}
		// Переходим к следующему ряду символов в терминале
		termRow++
	}

	// Сбрасываем стили и выстреливаем измененные пиксели
	writer.WriteString("\x1b[0m")
	writer.Flush()
}