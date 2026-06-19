package tuicanvas

import (
	"fmt"
	"os"
	"strings"
)

// readInput непрерывно считывает байты из стандартного ввода,
// распознает клавиши и события мыши, и отправляет их в соответствующие каналы.
func (s *Screen) readInput() {
	buf := make([]byte, 1024)
	for {
		// Чтение блокирует горутину, пока пользователь ничего не делает.
		// При Close() и отключении Raw Mode эта функция вернет ошибку, и мы чисто выйдем.
		n, err := os.Stdin.Read(buf) // Исправлена опечатка (было "Rea d")
		if err != nil {
			s.log(fmt.Sprintf("Ошибка чтения из stdin (возможно, терминал закрыт или сброшен): %v", err))
			return
		}

		// Проверяем, не пришел ли сигнал закрытия через done (на всякий случай)
		select {
		case <-s.done:
			s.log("Горутина readInput завершена по сигналу done")
			return
		default:
		}

		if n == 0 {
			continue
		}

		bytes := buf[:n]

		// 1. Обработка событий мыши SGR (начинаются с "\x1b[<")
		if n > 3 && bytes[0] == 27 && bytes[1] == '[' && bytes[2] == '<' { // Исправлены опечатки "& &"
			s.parseMouseEvent(string(bytes))
			continue
		}

		// 2. Обработка сложных клавиш (Escape-последовательности, например стрелочки)
		if bytes[0] == 27 && n > 1 { // Исправлена опечатка "& &"
			inputStr := string(bytes)
			keyName := ""
			switch inputStr {
			case "\x1b[A":
				keyName = "up"
			case "\x1b[B":
				keyName = "down"
			case "\x1b[C":
				keyName = "right"
			case "\x1b[D":
				keyName = "left"
			case "\x1b[1;2A":
				keyName = "shift+up"
			case "\x1b[1;2B":
				keyName = "shift+down"
			case "\x1b[1;2C":
				keyName = "shift+right"
			case "\x1b[1;2D":
				keyName = "shift+left"
			default:
				// Если последовательность неизвестна, но начинается с Esc, проверим на одиночный Esc
				if n == 1 {
					keyName = "escape"
				} else {
					// Логируем неизвестные последовательности для последующего добавления поддержки
					s.log(fmt.Sprintf("Неизвестная escape-последовательность: %q", inputStr))
				}
			}

			if keyName != "" {
				s.sendKey(KeyEvent{Key: keyName})
			}
			continue
		}

		// 3. Обработка сочетаний с Ctrl (в Raw Mode байты 1..26 соответствуют Ctrl+A..Ctrl+Z)
		if bytes[0] >= 1 && bytes[0] <= 26 && bytes[0] != 13 && bytes[0] != 9 { // Исправлены опечатки "& &"
			// Исключаем байт 13 (Enter) и 9 (Tab)
			letter := string('a' + bytes[0] - 1)
			s.sendKey(KeyEvent{Key: "ctrl+" + letter})
			continue
		}

		// 4. Обработка обычных одиночных клавиш
		var keyName string
		switch bytes[0] {
		case 27:
			keyName = "escape"
		case 13, 10:
			keyName = "enter"
		case 32:
			keyName = "space"
		case 127, 8:
			keyName = "backspace"
		case 9:
			keyName = "tab"
		default:
			// Обычный символ (символы могут быть многобайтными, например UTF-8, преобразуем весь срез)
			rawKey := strings.ToLower(string(bytes))

			rusToEng := map[string]string{
				"й": "q", "ц": "w", "у": "e", "к": "r", "е": "t", "н": "y",
				"ф": "a", "ы": "s", "в": "d", "а": "f", "п": "g", "р": "h",
				"я": "z", "ч": "x", "с": "c", "м": "v", "и": "b", "т": "n",
			}

			// Если символ есть в словаре — подменяем его на английский аналог
			if engKey, found := rusToEng[rawKey]; found {
				keyName = engKey
			} else {
				keyName = rawKey
			}
		}

		s.sendKey(KeyEvent{Key: keyName}) // Исправлена опечатка (было "ke yName")
	}
}

// sendKey безопасно отправляет событие клавиатуры в буферизированный канал
func (s *Screen) sendKey(ev KeyEvent) {
	select {
	case s.keyChan <- ev:
	default:
		// Если буфер канала (128) переполнен, игнорируем событие во избежание зависания
		s.log(fmt.Sprintf("Буфер канала клавиш переполнен, событие Key=%q пропущено", ev.Key))
	}
}

// parseMouseEvent декодирует строку формата SGR "\x1b[<button>;<x>;<y>M" (или m)
func (s *Screen) parseMouseEvent(mouseStr string) {
	// Убираем префикс "\x1b[<"
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
		s.log(fmt.Sprintf("Ошибка парсинга события мыши '%s': ожидалось 3 части, получено %d", mouseStr, len(parts)))
		return
	}

	var btnRaw, termX, termY int
	_, err1 := fmt.Sscanf(parts[0], "%d", &btnRaw)
	_, err2 := fmt.Sscanf(parts[1], "%d", &termX)
	_, err3 := fmt.Sscanf(parts[2], "%d", &termY)
	if err1 != nil || err2 != nil || err3 != nil {
		s.log(fmt.Sprintf("Ошибка парсинга координат мыши '%s': err1=%v, err2=%v, err3=%v", mouseStr, err1, err2, err3))
		return
	}

	// Координаты терминала начинаются с 1, переводим в индексы с 0
	xIdx := uint(termX - 1)
	yRow := uint(termY - 1)

	// Терминал присылает Y сверху вниз (строки).
	// Нам нужно пересчитать Y под пиксели нашего холста Canvas.
	// Одна строка терминала по высоте равна двум пикселям холста.
	termHeight := s.canvas.Height() / 2
	if yRow >= termHeight {
		s.log(fmt.Sprintf("Событие мыши отклонено: Y-строка (%d) вне диапазона холста (%d)", yRow, termHeight))
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
	fx, fy := s.canvas.GetCoords(xIdx, yIdx)

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
		// Для отладки можно раскомментировать следующую строку, но при активном движении мыши это создаст много записей:
		// s.log(fmt.Sprintf("Мышь: X=%d, Y=%d, Button=%v, IsDown=%v", ev.X, ev.Y, ev.Button, ev.IsDown))
	default:
		s.log("Событие мыши пропущено: канал mouseChan переполнен")
	}
}
