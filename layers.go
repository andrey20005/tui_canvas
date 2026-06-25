package tui_canvas

type Layers struct {
	canvas    *Canvas
	textLayer *TextLayer
	frame     *frame
}

func NewLayers() *Layers {
	return &Layers{}
}

func (l *Layers) acquireFrame(f *frame) {
	l.frame = f
	f.clear()

	// Вычисляем нужные размеры для слоёв
	d := BlockRenderer{}.density()
	canvasW := f.columns * uint(d)
	canvasH := f.rows * uint(d) * 2

	if l.canvas == nil {
		l.canvas = NewCanvas(canvasW, canvasH)
	} else {
		l.canvas.resize(canvasW, canvasH)
	}

	if l.textLayer == nil {
		l.textLayer = NewTextLayer(f.columns, f.rows)
	} else {
		l.textLayer.Resize(f.columns, f.rows)
	}
}

func (l *Layers) returnFrame() *frame {
	f := l.frame
	l.frame = nil
	return f
}

func (l *Layers) Canvas() *Canvas {
	return l.canvas
}

func (l *Layers) Text() *TextLayer {
	return l.textLayer
}

func (l *Layers) RenderLayers() {
	if l.frame == nil {
		return
	}
	BlockRenderer{}.render(l.canvas, l.frame)
	l.textLayer.renderIntoFrame(l.frame)
}
