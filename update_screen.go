package tuicanvas

import (
	"bufio"
	"os"
	"strconv"
)

// Update — главный диспетчер отрисовки.
func (s *Screen) Update() {
	s.backFrame.CopyFrom(s.frozenFrame)
	s.frozenFrame.BuildFrom(s.canvas, s.textLayer)

	if s.backFrame.Width() != s.frozenFrame.Width() || s.backFrame.Height() != s.frozenFrame.Height() {
		s.simpleUpdate()
	} else {
		s.optimizedUpdate()
	}
}

// simpleUpdate производит полную замену всех символов на экране терминала.
// Использует естественный перенос строки \n вместо ручного позиционирования курсора.
func (s *Screen) simpleUpdate() {
	w := s.frozenFrame.Width()
	h := s.frozenFrame.Height()
	if w == 0 || h == 0 {
		return
	}

	// Переносим курсор в абсолютное начало координат терминала (1,1)
	os.Stdout.WriteString("\x1b[H")

	// Накапливаем весь кадр в памяти
	writer := bufio.NewWriterSize(os.Stdout, 256*1024)

	for y := int(h) - 1; y >= 0; y-- {
		writer.WriteString("\x1b[")
		writer.WriteString(strconv.Itoa(int(h) - y))
		writer.WriteString(";1H")
		for x := uint(0); x < w; x++ {
			cell := s.frozenFrame.cells[y][x]
			topRGB := cell.Fg.ToRGB()
			bottomRGB := cell.Bg.ToRGB()

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
			writer.WriteString("m")
			writer.WriteString(string(cell.Rune))
		}
		writer.WriteString("\n")
	}

	writer.WriteString("\x1b[0m")
	writer.Flush()
}

// optimizedUpdate посимвольно сравнивает текущий снимок экрана с предыдущим.
// Здесь мы используем точечные прыжки, так как обновляются только измененные ячейки.
func (s *Screen) optimizedUpdate() {
	w := s.frozenFrame.Width()
	h := s.frozenFrame.Height()
	if w == 0 || h == 0 {
		return
	}

	writer := bufio.NewWriterSize(os.Stdout, 256*1024)

	cursorX := -1
	cursorY := -1

	for y := int(h) - 1; y >= 0; y-- {
		// Вычисляем физическую строку терминала для текущего y
		termRow := int(h) - y

		for x := uint(0); x < w; x++ {
			currentCell := s.frozenFrame.cells[y][x]
			backCell := s.backFrame.cells[y][x]

			if currentCell == backCell {
				continue
			}

			// Вычисляем физическую колонку терминала (индексы с 1)
			termCol := int(x + 1)

			if termRow != cursorY || termCol != cursorX {
				writer.WriteString("\x1b[")
				writer.WriteString(strconv.Itoa(termRow))
				writer.WriteString(";")
				writer.WriteString(strconv.Itoa(termCol))
				writer.WriteString("H")
				
				// Синхронизируем наше знание о позиции курсора
				cursorY = termRow
				cursorX = termCol + 1 // +1 после печати
			}

			topRGB := currentCell.Fg.ToRGB()
			bottomRGB := currentCell.Bg.ToRGB()

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
			writer.WriteString("m")
			writer.WriteString(string(currentCell.Rune))
		}
	}

	writer.WriteString("\x1b[0m")
	writer.Flush()
}
