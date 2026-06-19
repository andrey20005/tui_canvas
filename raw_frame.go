package tuicanvas

import "fmt"

// TerminalCell хранит состояние одного знакоместа терминала.
type TerminalCell struct {
	Rune rune  // Символ ('▀' или буква из TextLayer)
	Fg   Color // Финальный цвет текста
	Bg   Color // Финальный цвет фона
}

// RawFrame — промежуточный плоский слепок экрана.
type RawFrame struct {
	cells  [][]TerminalCell // строчка с индексом 0 нижняя 
	width  uint
	height uint
}

// NewRawFrame создает пустой слепок заданного размера.
func NewRawFrame(w, h uint) *RawFrame {
	data := make([][]TerminalCell, h)
	for y := uint(0); y < h; y++ {
		data[y] = make([]TerminalCell, w)
	}
	return &RawFrame{cells: data, width: w, height: h}
}

// BuildFrom склеивает графику Canvas и текст TextLayer в единый плоский кадр.
// Если размеры слоев не совпадают — жестко паникует, так как это внутренний баг движка.
// Метод автоматически пересоздает свою матрицу, если размеры терминала изменились.
func (rf *RawFrame) BuildFrom(canvas *Canvas, textLayer *TextLayer) {
	// Жесткая валидация размеров 
	if canvas.Width() != textLayer.Width() || canvas.Height()/2 != textLayer.Height() {
		panic(fmt.Sprintf(
			"tuicanvas bug: dimensions mismatch. Canvas: %dx%d, TextLayer: %dx%d",
			canvas.Width(), canvas.Height(), textLayer.Width(), textLayer.Height(),
		))
	}

	w := canvas.Width()
	h := textLayer.Height() // Высота в строках терминала

	//  Если размеры терминала изменились, перестраиваем свою матрицу на лету
	if rf.width != w || rf.height != h {
		rf.cells = make([][]TerminalCell, h)
		for y := uint(0); y < h; y++ {
			rf.cells[y] = make([]TerminalCell, w)
		}
		rf.width = w
		rf.height = h
	}

	// Сборка кадра
	for y := uint(0); y < h; y++ {
		yBottom := y * 2
		yTop := yBottom + 1

		for x := uint(0); x < w; x++ {
			cellText := textLayer.data[y][x]

			if cellText.Is && cellText.Shader != nil {
				topColor := canvas.data[yTop][x]
				bottomColor := canvas.data[yBottom][x]

				finalRune, fg, bg := cellText.Shader.Process(cellText.Rune, topColor, bottomColor)
				rf.cells[y][x] = TerminalCell{Rune: finalRune, Fg: fg, Bg: bg}
			} else {
				rf.cells[y][x] = TerminalCell{
					Rune: '▀',
					Fg:   canvas.data[yTop][x],
					Bg:   canvas.data[yBottom][x],
				}
			}
		}
	}
}

// CopyFrom полностью копирует содержимое другого кадра (нужно для сохранения back-буфера).
func (rf *RawFrame) CopyFrom(src *RawFrame) {
	if rf == nil || src == nil {
		return
	}

	// Синхронизируем размеры, если они отличаются
	if rf.width != src.width || rf.height != src.height {
		rf.width = src.width
		rf.height = src.height
		rf.cells = make([][]TerminalCell, rf.height)
		for y := uint(0); y < rf.height; y++ {
			rf.cells[y] = make([]TerminalCell, rf.width)
		}
	}

	// Быстро копируем строки через встроенный copy
	for y := uint(0); y < rf.height; y++ {
		copy(rf.cells[y], src.cells[y])
	}
}

// Width возвращает ширину кадра
func (rf *RawFrame) Width() uint { return rf.width }

// Height возвращает высоту кадра
func (rf *RawFrame) Height() uint { return rf.height }
