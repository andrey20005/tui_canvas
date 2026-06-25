package tui_canvas

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sync"

	"golang.org/x/term"
)

// Screen управляет состоянием терминала, обрабатывает ввод и ресайз,
// а также координирует графический и текстовый слои.
type Screen struct {
	// Актуальные размеры терминала
	rows, columns uint

	// слои для рисования
	layers *Layers

	// Каналы для событий
	keyChan    chan KeyEvent
	mouseChan  chan MouseEvent
	resizeChan chan ResizeEvent

	// Внутренний канал для обработки ресайза библиотекой
	internalResizeChan chan ResizeEvent

	done chan struct{}

	// Рендерер
	displayChan chan *frame
	renderWG    sync.WaitGroup
	framePool   sync.Pool

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
		layers: NewLayers(),
		rows:               uint(h),
		columns:            uint(w),
		keyChan:            make(chan KeyEvent, 128),
		mouseChan:          make(chan MouseEvent, 256),
		resizeChan:         make(chan ResizeEvent, 16),
		internalResizeChan: make(chan ResizeEvent, 16),
		done:               make(chan struct{}),
		displayChan:        make(chan *frame, 1),
		oldState:           oldState,
		framePool: sync.Pool{
			New: func() interface{} {
				return NewFrame(uint(w), uint(h))
			},
		},
	}
	s.layers.acquireFrame(NewFrame(s.columns, s.rows))

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
	go displayLoop(s.displayChan, &s.framePool, s.done, s.log, &s.renderWG)
	go watchResizeLoop(s.done, s.internalResizeChan)
	go readInputLoop(s.keyChan, s.mouseChan, s.done, s.log, func() uint { return s.rows })

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

// Display подготавливает кадр и отправляет его в displayLoop.
func (s *Screen) Display() {
	frameToView := s.layers.returnFrame()

	select {
	case s.displayChan <- frameToView:
	default:
		// Дропаем старый
		oldFrame := <-s.displayChan
		s.framePool.Put(oldFrame)
		s.displayChan <- frameToView
	}

	// проверяем не изменились ли размеры экрана
	s.handleResize()
	
	nextFrame := s.framePool.Get().(*frame)
	if nextFrame.columns != s.columns || nextFrame.rows != s.rows {
		nextFrame = NewFrame(s.columns, s.rows)
	}
	s.layers.acquireFrame(nextFrame)
}

func (s *Screen) handleResize() {
	select {
	case ev := <-s.internalResizeChan:
		// нужно что-то еще вызвать тут
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

func (s *Screen) KeyEvents() <-chan KeyEvent       { return s.keyChan }
func (s *Screen) MouseEvents() <-chan MouseEvent   { return s.mouseChan }
func (s *Screen) ResizeEvents() <-chan ResizeEvent { return s.resizeChan }
func (s *Screen) Layers() *Layers                   { return s.layers }
