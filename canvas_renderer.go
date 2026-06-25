package tui_canvas

type canvasRenderer interface {
	render(canvas *Canvas, frame *frame)
	density() int // сколько пикселей canvas на одну ячейку в ширину
}

type BlockRenderer struct{}

func (BlockRenderer) render(canvas *Canvas, frame *frame) {
	for y := uint(0); y < frame.rows; y++ {
		for x := uint(0); x < frame.columns; x++ {
			topColor := canvas.data[y*2][x]
			bottomColor := canvas.data[y*2+1][x]
			frame.cells[y][x] = terminalCell{
				Rune: '▀',
				Fg:   bottomColor,
				Bg:   topColor,
			}
		}
	}
}

func (BlockRenderer) density() int {
	return 1
}
