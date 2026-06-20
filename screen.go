package tuicanvas

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"golang.org/x/term"
)

// Screen управляет состоянием терминала, обрабатывает ввод и ресайз,
// а также координирует графический и текстовый слои.
type Screen struct {
	canvas    *Canvas
	textLayer *TextLayer

	// Актуальные размеры
	rows, columns uint

	// Каналы для событий
	keyChan    chan KeyEvent
	mouseChan  chan MouseEvent
	resizeChan chan ResizeEvent

	// Внутренний канал для обработки ресайза библиотекой
	internalResizeChan chan ResizeEvent

	done chan struct{}

	// Рендерер
	renderChan chan *RawFrame
	renderWG   sync.WaitGroup
	framePool  sync.Pool

	oldState *term.State
	logger   *log.Logger
}

// NewScreen инициализирует и перехватывает терминал, создавая менеджер экрана.
func NewScreen(logPath string) (*Screen, error) {
	fdOut := int(os.Stdout.Fd())
	fdIn := int(os.Stdin.Fd())
	if !term.IsTerminal(fdOut) || !term.IsTerminal(fdIn) {
		return nil, fmt.Errorf("standard input/output is not a terminal")
	}

	w, h, err := term.GetSize(fdOut)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size: %w", err)
	}

	oldState, err := term.MakeRaw(fdIn)
	if err != nil {
		return nil, fmt.Errorf("failed to enable raw mode: %w", err)
	}

	s := &Screen{
		canvas:             NewCanvas(uint(w), uint(h*2)),
		textLayer:          NewTextLayer(uint(w), uint(h)),
		rows:               uint(h),
		columns:            uint(w),
		keyChan:            make(chan KeyEvent, 128),
		mouseChan:          make(chan MouseEvent, 256),
		resizeChan:         make(chan ResizeEvent, 16),
		internalResizeChan: make(chan ResizeEvent, 16),
		done:               make(chan struct{}),
		renderChan:         make(chan *RawFrame, 1),
		oldState:           oldState,
		framePool: sync.Pool{
			New: func() interface{} {
				return NewRawFrame(uint(w), uint(h))
			},
		},
	}

	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			s.logger = log.New(file, "[SCREEN] ", log.Ltime|log.Lshortfile)
		}
	}

	writer := bufio.NewWriter(os.Stdout)
	writer.WriteString("\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h")
	writer.Flush()

	s.renderWG.Add(1)
	go renderLoop(s.renderChan, &s.framePool, s.done, s.log, &s.renderWG)
	go s.watchResize()
	go readInput(s.keyChan, s.mouseChan, s.done, s.log, func() uint { return s.canvas.Height() })

	return s, nil
}

func (s *Screen) log(msg string) {
	if s.logger != nil {
		s.logger.Println(msg)
	}
}

// Close безопасно закрывает экран, останавливает горутины и возвращает настройки терминала обратно.
func (s *Screen) Close() {
	close(s.done)
	s.renderWG.Wait()

	writer := bufio.NewWriter(os.Stdout)
	writer.WriteString("\x1b[?1003l\x1b[?1006l\x1b[?25h\x1b[?1049l")
	writer.Flush()

	if s.oldState != nil {
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
}

// Draw выполняет отрисовку callback-функцией и отправляет результат в рендерер.
func (s *Screen) Draw(f func(canvas *Canvas, textLayer *TextLayer)) {
	s.handleResize()
	f(s.canvas, s.textLayer)
	s.Update()
}

// Update подготавливает кадр и отправляет его в рендерер.
func (s *Screen) Update() {
	s.handleResize()

	// 1. Берем кадр из пула
	frame := s.framePool.Get().(*RawFrame)

	// 2. BuildFrom требует согласованности размеров, проверяем
	if frame.Width() != s.columns || frame.Height() != s.rows {
		// Если размер пула не совпал с текущим экраном, создаем новый
		frame = NewRawFrame(s.columns, s.rows)
	}

	// 3. Строим кадр
	frame.BuildFrom(s.canvas, s.textLayer)

	// 4. Отправляем в рендерер
	select {
	case s.renderChan <- frame:
	default:
		// Дропаем старый
		oldFrame := <-s.renderChan
		s.framePool.Put(oldFrame)
		s.renderChan <- frame
	}
}

func (s *Screen) handleResize() {
	select {
	case ev := <-s.internalResizeChan:
		s.canvas.resize(ev.Width, ev.Height*2)
		s.textLayer.Resize(ev.Width, ev.Height)
		s.rows = ev.Height
		s.columns = ev.Width

		// Оповещаем пользователя
		select {
		case s.resizeChan <- ev:
		default:
		}
	default:
	}
}

func (s *Screen) Canvas() *Canvas                  { return s.canvas }
func (s *Screen) TextLayer() *TextLayer            { return s.textLayer }
func (s *Screen) KeyEvents() <-chan KeyEvent       { return s.keyChan }
func (s *Screen) MouseEvents() <-chan MouseEvent   { return s.mouseChan }
func (s *Screen) ResizeEvents() <-chan ResizeEvent { return s.resizeChan }

func (s *Screen) watchResize() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-s.done:
			return
		case <-sigChan:
			w, h, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				select {
				case s.internalResizeChan <- ResizeEvent{Width: uint(w), Height: uint(h)}:
				default:
				}
			}
		}
	}
}
