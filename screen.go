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
	// Активные слои, с которыми работает пользователь
	canvas    *Canvas
	textLayer *TextLayer

	// Промежуточные плоские кадры для Double Buffering
	frozenFrame *RawFrame
	backFrame   *RawFrame

	// Центральный замок потокобезопасности для Canvas и TextLayer
	mu sync.Mutex

	// Внутренние каналы для событий
	keyChan    chan KeyEvent
	mouseChan  chan MouseEvent
	resizeChan chan ResizeEvent

	// Канал для безопасного завершения фоновых горутин
	done chan struct{}

	// Хранилище старого состояния терминала для восстановления при Close()
	oldState *term.State

	// Опциональный логгер для отладки
	logger *log.Logger
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

	// Размеры в пикселях и строках терминала
	canvasW := uint(w)
	canvasH := uint(h * 2)
	termW := uint(w)
	termH := uint(h)

	oldState, err := term.MakeRaw(fdIn)
	if err != nil {
		return nil, fmt.Errorf("failed to enable raw mode: %w", err)
	}

	s := &Screen{
		canvas:      NewCanvas(canvasW, canvasH),
		textLayer:   NewTextLayer(termW, termH),
		frozenFrame: NewRawFrame(termW, termH),
		backFrame:   NewRawFrame(termW, termH),
		keyChan:     make(chan KeyEvent, 128),
		mouseChan:   make(chan MouseEvent, 256),
		resizeChan:  make(chan ResizeEvent),
		done:        make(chan struct{}),
		oldState:    oldState,
	}

	// Настраиваем опциональный логгер
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[SCREEN] Не удалось открыть файл логов %s: %v\n", logPath, err)
		} else {
			s.logger = log.New(file, "[SCREEN] ", log.Ltime|log.Lshortfile)
			s.log("--- Логирование экрана запущено ---")
			s.log(fmt.Sprintf("Стартовый размер терминала: %dx%d (Canvas: %dx%d, Text: %dx%d)", w, h, canvasW, canvasH, termW, termH))
		}
	}

	writer := bufio.NewWriter(os.Stdout)
	writer.WriteString("\x1b[?1049h\x1b[?25l\x1b[?1003h\x1b[?1006h")
	writer.Flush()

	go s.watchResize()
	go s.readInput()

	return s, nil
}

// Вспомогательный приватный метод для безопасной записи в лог
func (s *Screen) log(msg string) {
	if s.logger != nil {
		s.logger.Println(msg)
	}
}

// Close безопасно закрывает экран, останавливает горутины и возвращает настройки терминала обратно.
func (s *Screen) Close() {
	s.log("Начало безопасного закрытия экрана...")

	// Сигнализируем фоновым горутинам, что пора завершать работу
	close(s.done)

	// Отправляем восстанавливающие ANSI-коды:
	// \x1b[?1003l\x1b[?1006l - отключить захват мыши
	// \x1b[?25h             - вернуть стандартный курсор
	// \x1b[?1049l            - вернуться в основной буфер терминала (восстановить историю консоли)
	writer := bufio.NewWriter(os.Stdout)
	writer.WriteString("\x1b[?1003l\x1b[?1006l\x1b[?25h\x1b[?1049l")
	writer.Flush()

	// Восстанавливаем канонический режим ввода терминала (Echo, Enter)
	if s.oldState != nil {
		err := term.Restore(int(os.Stdin.Fd()), s.oldState)
		if err != nil {
			s.log(fmt.Sprintf("ОШИБКА восстановления состояния терминала: %v", err))
		} else {
			s.log("Состояние терминала успешно восстановлено")
		}
	}
	s.log("Экран успешно закрыт")
}

// ==========================================
// ГЕТТЕРЫ КАНАЛОВ И ХОЛСТА (Публичный API)
// ==========================================

// Canvas возвращает указатель на текущий холст для рисования.
func (s *Screen) Canvas() *Canvas {
	return s.canvas
}

func (s *Screen) TextLayer() *TextLayer {
	return s.textLayer
}

// KeyEvents возвращает канал событий клавиатуры только для чтения.
func (s *Screen) KeyEvents() <-chan KeyEvent {
	return s.keyChan
}

// MouseEvents возвращает канал событий мыши только для чтения.
func (s *Screen) MouseEvents() <-chan MouseEvent {
	return s.mouseChan
}

// ResizeEvents возвращает неблокирующий канал событий изменения размера только для чтения.
func (s *Screen) ResizeEvents() <-chan ResizeEvent {
	return s.resizeChan
}

// Защищает состояние холста и текстового слоя.
func (s *Screen) Lock() {
	s.mu.Lock()
}

// Unlock открывает доступ к холсту и текстовому слою.
func (s *Screen) Unlock() {
	s.mu.Unlock()
}

// watchResize отслеживает изменение размеров окна терминала от ОС.
func (s *Screen) watchResize() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)

	for {
		select {
		case <-s.done:
			s.log("Горутина watchResize завершена")
			return

		case <-sigChan:
			fdOut := int(os.Stdout.Fd())
			w, h, err := term.GetSize(fdOut)
			if err != nil {
				s.log(fmt.Sprintf("Ошибка получения размера: %v", err))
				continue
			}

			canvasW, canvasH := uint(w), uint(h*2)
			termW, termH := uint(w), uint(h)

			s.mu.Lock()
			s.canvas.Resize(canvasW, canvasH)
			s.textLayer.Resize(termW, termH)
			s.mu.Unlock()

			s.log(fmt.Sprintf("Ресайз в памяти: Canvas %dx%d, TextLayer %dx%d", canvasW, canvasH, termW, termH))

			select {
			case s.resizeChan <- ResizeEvent{Width: canvasW, Height: canvasH}:
			default:
				s.log("Событие ресайза пропущено (канал занят)")
			}
		}
	}
}
