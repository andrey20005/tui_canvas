package tui_canvas

import (
	"bufio"
	"os"
	"strconv"
	"sync"
)

// renderLoop запускается в отдельной горутине, монопольно владеет stdout
// и выводит кадры по мере их поступления.
func renderLoop(
	renderChan <-chan *RawFrame,
	framePool *sync.Pool,
	done <-chan struct{},
	logFunc func(string),
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	var lastFrame *RawFrame
	writer := bufio.NewWriterSize(os.Stdout, 256*1024)

	for {
		select {
		case <-done:
			logFunc("Горутина renderLoop завершена")
			if lastFrame != nil {
				framePool.Put(lastFrame)
			}
			return
		case frame := <-renderChan:
			render(writer, lastFrame, frame)
			writer.Flush()
			
			// Возвращаем старый кадр в пул, а текущий запоминаем как последний
			if lastFrame != nil {
				framePool.Put(lastFrame)
			}
			lastFrame = frame
		}
	}
}

// render выбирает стратегию отрисовки
func render(writer *bufio.Writer, lastFrame, currentFrame *RawFrame) {
	if lastFrame == nil || 
	   lastFrame.Width() != currentFrame.Width() || 
	   lastFrame.Height() != currentFrame.Height() {
		simpleRender(writer, currentFrame)
	} else {
		optimizedRender(writer, lastFrame, currentFrame)
	}
}

// simpleRender производит полную перерисовку
func simpleRender(writer *bufio.Writer, frame *RawFrame) {
	w, h := frame.Width(), frame.Height()
	writer.WriteString("\x1b[H")

	for y := int(h) - 1; y >= 0; y-- {
		writer.WriteString("\x1b[")
		writer.WriteString(strconv.Itoa(int(h)-y))
		writer.WriteString(";1H")
		for x := uint(0); x < w; x++ {
			writeCell(writer, frame.cells[y][x])
		}
	}
	writer.WriteString("\x1b[0m")
}

// optimizedRender обновляет только изменившиеся ячейки
func optimizedRender(writer *bufio.Writer, lastFrame, currentFrame *RawFrame) {
	w, h := currentFrame.Width(), currentFrame.Height()
	cursorX, cursorY := -1, -1

	for y := int(h) - 1; y >= 0; y-- {
		termRow := int(h) - y
		for x := uint(0); x < w; x++ {
			currentCell := currentFrame.cells[y][x]
			backCell := lastFrame.cells[y][x]

			if currentCell == backCell {
				continue
			}

			termCol := int(x + 1)
			if termRow != cursorY || termCol != cursorX {
				writer.WriteString("\x1b[")
				writer.WriteString(strconv.Itoa(termRow))
				writer.WriteString(";")
				writer.WriteString(strconv.Itoa(termCol))
				writer.WriteString("H")
				cursorY, cursorX = termRow, termCol+1
			}
			writeCell(writer, currentCell)
		}
	}
	writer.WriteString("\x1b[0m")
}

// writeCell записывает цвета и символ ячейки
func writeCell(writer *bufio.Writer, cell TerminalCell) {
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
