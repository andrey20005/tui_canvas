//go:build windows

package tui_canvas

import (
	"os"
	"time"

	"golang.org/x/term"
)

// watchResizeWindows опрашивает размер консоли через интервалы времени
func watchResize(done chan struct{}, resizeChan chan ResizeEvent) {
	lastW, lastH, _ := term.GetSize(int(os.Stdout.Fd()))

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			w, h, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				continue
			}

			if w != lastW || h != lastH {
				lastW = w
				lastH = h

				select {
				case resizeChan <- ResizeEvent{Width: uint(w), Height: uint(h)}:
				default:
				}
			}
		}
	}
}
