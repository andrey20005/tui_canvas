package tuicanvas

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// Screen управляет состоянием терминала, обрабатывает ввод и ресайз,
// а также предоставляет холст для рисования.
type Screen struct {
	// Графические буферы
	currentCanvas *Canvas // Холст, на котором рисует пользователь прямо сейчас
	frozenCanvas *Canvas
	backCanvas    *Canvas // Копия предыдущего кадра для оптимизации вывода (Double Buffering)

	// Внутренние двунаправленные каналы для событий
	keyChan    chan KeyEvent
	mouseChan  chan MouseEvent
	resizeChan chan ResizeEvent

	// Канал для безопасного завершения фоновых горутин
	done chan struct{}

	// Хранилище старого состояния терминала для восстановления при Close()
	oldState *term.State

	logger *log.Logger
}

// NewScreen инициализирует экран.
// Если logPath не пустой (например, "debug.log"), включает детальное логирование в этот файл.
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

	canvasW := uint(w)
	canvasH := uint(h * 2)

	oldState, err := term.MakeRaw(fdIn)
	if err != nil {
		return nil, fmt.Errorf("failed to enable raw mode: %w", err)
	}

	s := &Screen{
		currentCanvas: NewCanvas(canvasW, canvasH),
		backCanvas:    NewCanvas(canvasW, canvasH),
		frozenCanvas:  NewCanvas(canvasW, canvasH),
		keyChan:       make(chan KeyEvent, 128),
		mouseChan:     make(chan MouseEvent, 256), // Исправлена опечатка (было "2 56")
		resizeChan:    make(chan ResizeEvent),
		done:          make(chan struct{}),
		oldState:      oldState,
	}

	// Настраиваем опциональный логгер
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			// Логгер еще не создан, поэтому пишем критическую ошибку напрямую в stderr
			fmt.Fprintf(os.Stderr, "[SCREEN] Не удалось открыть файл логов %s: %v\n", logPath, err)
		} else {
			// Создаем изолированный логгер, который пишет только в этот файл
			s.logger = log.New(file, "[SCREEN] ", log.Ltime|log.Lshortfile)
			s.log("--- Логирование экрана запущено ---")
			s.log(fmt.Sprintf("Стартовый размер терминала: %dx%d (Canvas: %dx%d)", w, h, canvasW, canvasH))
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
	return s.currentCanvas
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

// watchResize отслеживает сигналы изменения размера терминала от ОС.
// Запускается как фоновая горутина.
func (s *Screen) watchResize() {
	// 1. Создаем буферизированный канал для приема сигналов от ОС
	sigChan := make(chan os.Signal, 1)
	// 2. Регистрируем канал на получение сигнала изменения размера окна (SIGWINCH)
	signal.Notify(sigChan, syscall.SIGWINCH)

	// При завершении горутины снимаем регистрацию сигнала
	defer signal.Stop(sigChan)

	for {
		select {
		case <-s.done:
			// Метод Close() закрыл канал done — мягко завершаем горутину
			s.log("Горутина watchResize завершена")
			return

		case <-sigChan:
			// ОС сообщила, что размер окна изменился!
			s.log("Получен сигнал SIGWINCH (изменение размера окна)")
			fdOut := int(os.Stdout.Fd())

			w, h, err := term.GetSize(fdOut)
			if err != nil {
				s.log(fmt.Sprintf("Не удалось прочитать новый размер терминала: %v", err))
				continue // Если не удалось прочитать размер, игнорируем этот тик
			}

			canvasW := uint(w)
			canvasH := uint(h * 2)

			// 3. Автоматически изменяем размеры наших внутренних буферов холста.
			s.currentCanvas.Resize(canvasW, canvasH)
			s.backCanvas.Resize(canvasW, canvasH)
			s.log(fmt.Sprintf("Размер холста изменен на: %dx%d", canvasW, canvasH))

			// 4. Формируем событие для пользователя
			ev := ResizeEvent{
				Width:  canvasW,
				Height: canvasH,
			}

			// 5. Неблокирующая отправка в канал:
			select {
			case s.resizeChan <- ev:
				// Успешно передали событие в приложение
			default:
				s.log("Событие изменения размера пропущено: канал resizeChan переполнен")
			}
		}
	}
}

