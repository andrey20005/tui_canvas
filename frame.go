package tui_canvas

// terminalCell хранит состояние одного знакоместа терминала.
// Ось y везде направленна вверх, строчка с нулевым индексом нижняя
type terminalCell struct {
	Rune rune // Символ ('▀' или буква из TextLayer)
	Fg   RGB  // цвет символа
	Bg   RGB  // цвет фона
}

// frame — промежуточный плоский слепок экрана.
type frame struct {
	cells   [][]terminalCell // строчка с индексом 0 нижняя
	columns uint
	rows    uint
}

// NewFrame создает пустой слепок заданного размера.
func NewFrame(c, r uint) *frame {
	data := make([][]terminalCell, r)
	for y := uint(0); y < r; y++ {
		data[y] = make([]terminalCell, c)
	}
	return &frame{cells: data, columns: c, rows: r}
}

func (f *frame) clear() {
	empty := terminalCell{}
	for x := range f.cells[0] {
		f.cells[0][x] = empty
	}
	for y := 1; uint(y) < f.rows; y++ {
		copy(f.cells[y], f.cells[0])
	}
}

// Width возвращает ширину кадра
func (rf *frame) Width() uint { return rf.columns }

// Height возвращает высоту кадра
func (rf *frame) Height() uint { return rf.rows }
