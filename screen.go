package tuicanvas

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// Screen управляет состоянием терминала, обрабатывает ввод и ресайз,
// а также предоставляет холст для рисования.
type Screen struct {
	// Графические буферы
	currentCanvas *Canvas // Холст, на котором рисует пользователь прямо сейчас
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

/// NewScreen инициализирует экран. 
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
		keyChan:       make(chan KeyEvent, 128),
		mouseChan:     make(chan MouseEvent, 256),
		resizeChan:    make(chan ResizeEvent),
		done:          make(chan struct{}),
		oldState:      oldState,
	}

	// Настраиваем опциональный логгер
	if logPath != "" {
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
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
		term.Restore(int(os.Stdin.Fd()), s.oldState)
	}
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

// Update временно просто возвращает курсор в начало экрана и перерисовывает весь холст.
// (В будущем здесь будет оптимизированное посимвольное сравнение кадров).
func (s *Screen) Update() {
	os.Stdout.WriteString("\x1b[H")
	s.currentCanvas.Render()
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
			return

		case <-sigChan:
			// ОС сообщила, что размер окна изменился!
			fdOut := int(os.Stdout.Fd())
			
			w, h, err := term.GetSize(fdOut)
			if err != nil {
				continue // Если не удалось прочитать размер, игнорируем этот тик
			}

			canvasW := uint(w)
			canvasH := uint(h * 2)

			// 3. Автоматически изменяем размеры наших внутренних буферов холста.
			// Метод Resize, который мы написали ранее, сам отцентрирует картинку!
			s.currentCanvas.Resize(canvasW, canvasH)
			s.backCanvas.Resize(canvasW, canvasH)

			// 4. Формируем событие для пользователя
			ev := ResizeEvent{
				Width:  canvasW,
				Height: canvasH,
			}

			// 5. Неблокирующая отправка в канал:
			// Если главный цикл сейчас занят рендерингом тяжелого кадра,
			// промежуточное событие ресайза просто выбросится, чтобы не забивать память.
			select {
			case s.resizeChan <- ev:
				// Успешно передали событие в приложение
			default:
				// Канал занят — пропускаем этот промежуточный размер
			}
		}
	}
}

// readInput непрерывно считывает байты из стандартного ввода,
// распознает клавиши и события мыши, и отправляет их в соответствующие каналы.
func (s *Screen) readInput() {
	buf := make([]byte, 1024)

	for {
		// Чтение блокирует горутину, пока пользователь ничего не делает.
		// При Close() и отключении Raw Mode эта функция вернет ошибку, и мы чисто выйдем.
		n, err := os.Stdin.Read(buf)
		if err != nil {
			return
		}

		// Проверяем, не пришел ли сигнал закрытия через done (на всякий случай)
		select {
		case <-s.done:
			return
		default:
		}

		if n == 0 {
			continue
		}

		bytes := buf[:n]

		// 1. Обработка событий мыши SGR (начинаются с "\x1b[<")
		if n > 3 && bytes[0] == 27 && bytes[1] == '[' && bytes[2] == '<' {
			s.parseMouseEvent(string(bytes))
			continue
		}

		// 2. Обработка сложных клавиш (Escape-последовательности, например стрелочки)
		if bytes[0] == 27 && n > 1 {
			inputStr := string(bytes)
			keyName := ""
			switch inputStr {
			case "\x1b[A": keyName = "up"
			case "\x1b[B": keyName = "down"
			case "\x1b[C": keyName = "right"
			case "\x1b[D": keyName = "left"
			case "\x1b[1;2A": keyName = "shift+up"
			case "\x1b[1;2B": keyName = "shift+down"
			case "\x1b[1;2C": keyName = "shift+right"
			case "\x1b[1;2D": keyName = "shift+left"
			default:
				// Если последовательность неизвестна, но начинается с Esc, проверим на одиночный Esc
				if n == 1 {
					keyName = "escape"
				}
			}

			if keyName != "" {
				s.sendKey(KeyEvent{Key: keyName})
			}
			continue
		}

		// 3. Обработка сочетаний с Ctrl (в Raw Mode байты 1..26 соответствуют Ctrl+A..Ctrl+Z)
		if bytes[0] >= 1 && bytes[0] <= 26 && bytes[0] != 13 && bytes[0] != 9 { 
			// Исключаем байт 13 (Enter) и 9 (Tab)
			letter := string('a' + bytes[0] - 1)
			s.sendKey(KeyEvent{Key: "ctrl+" + letter})
			continue
		}

		// 4. Обработка обычных одиночных клавиш
		var keyName string
		switch bytes[0] {
		case 27: keyName = "escape"
		case 13, 10: keyName = "enter"
		case 32: keyName = "space"
		case 127, 8: keyName = "backspace"
		case 9: keyName = "tab"
		default:
			// Обычный символ (символы могут быть многобайтными, например UTF-8, преобразуем весь срез)
			keyName = strings.ToLower(string(bytes))
		}

		s.sendKey(KeyEvent{Key: keyName})
	}
}

// sendKey безопасно отправляет событие клавиатуры в буферизированный канал
func (s *Screen) sendKey(ev KeyEvent) {
	select {
	case s.keyChan <- ev:
	default:
		// Если буфер канала (128) переполнен, игнорируем событие во избежание зависания
	}
}

// parseMouseEvent декодирует строку формата SGR "\x1b[<button>;<x>;<y>M" (или m)
func (s *Screen) parseMouseEvent(mouseStr string) {
	// Убираем префикс префикс "\x1b[<"
	str := mouseStr[3:]
	
	// Определяем, нажата или отпущена кнопка (M — нажата/движение, m — отпущена)
	isDown := true
	if strings.HasSuffix(str, "m") {
		isDown = false
	}
	
	// Отрезаем последний символ ('M' или 'm')
	str = str[:len(str)-1]

	// Разбиваем строку по точке с запятой на компоненты: button, x, y
	parts := strings.Split(str, ";")
	if len(parts) != 3 {
		return
	}

	var btnRaw, termX, termY int
	_, err1 := fmt.Sscanf(parts[0], "%d", &btnRaw)
	_, err2 := fmt.Sscanf(parts[1], "%d", &termX)
	_, err3 := fmt.Sscanf(parts[2], "%d", &termY)
	if err1 != nil || err2 != nil || err3 != nil {
		return
	}

	// Координаты терминала начинаются с 1, переводим в индексы с 0
	xIdx := uint(termX - 1)
	yRow := uint(termY - 1)

	// Терминал присылает Y сверху вниз (строки).
	// Нам нужно пересчитать Y под пиксели нашего холста Canvas.
	// Одна строка терминала по высоте равна двум пикселям холста.
	termHeight := s.currentCanvas.Height() / 2
	if yRow >= termHeight {
		return
	}
	
	// Так как ось Y на холсте направлена ВВЕРХ, инвертируем индекс строки терминала
	// И умножаем на 2, чтобы получить верхний пиксель в этой строке
	yIdx := (termHeight - 1 - yRow) * 2

	// Определяем тип кнопки
	button := MouseButton(btnRaw & 67) // 67 маскирует биты скролла и стандартных кнопок
	
	// Проверяем бит движения мыши (если 32-й бит равен 1 — мышь просто двигалась)
	if (btnRaw & 32) != 0 {
		button = MouseMove
	}

	// Запрашиваем у Canvas вещественные координаты для этой точки холста
	fx, fy := s.currentCanvas.GetCoords(xIdx, yIdx)

	// Формируем событие мыши
	ev := MouseEvent{
		X:      xIdx,
		Y:      yIdx,
		FX:     fx,
		FY:     fy,
		Button: button,
		IsDown: isDown,
	}

	// Неблокирующая отправка в канал мыши (защита от лагов при быстром перемещении)
	select {
	case s.mouseChan <- ev:
	default:
	}
}

