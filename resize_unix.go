//go:build !windows

package tui_canvas

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// watchResizeUnix слушает системный сигнал SIGWINCH
func watchResize(done chan struct{}, resizeChan chan ResizeEvent) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-done:
			return
		case <-sigChan:
			w, h, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				select {
				case resizeChan <- ResizeEvent{Width: uint(w), Height: uint(h)}:
				default:
				}
			}
		}
	}
}
