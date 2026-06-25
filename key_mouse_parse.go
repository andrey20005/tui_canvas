package tui_canvas

import (
	"fmt"
	"os"
	"strings"
)

// readInputLoop непрерывно считывает байты из стандартного ввода,
// распознает клавиши и события мыши, и отправляет их в соответствующие каналы.
func readInputLoop(
	keyChan chan<- KeyEvent,
	mouseChan chan<- MouseEvent,
	done <-chan struct{},
	logFunc func(string),
	getFrameRows func() uint,
) {
	buf := make([]byte, 1024)
	for {
		// Чтение блокирует горутину, пока пользователь ничего не делает.
		// При Close() и отключении Raw Mode эта функция вернет ошибку, и мы чисто выйдем.
		n, err := os.Stdin.Read(buf)
		if err != nil {
			logFunc(fmt.Sprintf("Ошибка чтения из stdin (возможно, терминал закрыт или сброшен): %v", err))
			return
		}

		// Проверяем, не пришел ли сигнал закрытия через done
		select {
		case <-done:
			logFunc("Горутина readInput завершена по сигналу done")
			return
		default:
		}

		if n == 0 {
			continue
		}

		bytes := buf[:n]

		// Обработка событий мыши SGR (начинаются с "\x1b[<")
		if n > 3 && bytes[0] == 27 && bytes[1] == '[' && bytes[2] == '<' {
			if ev, ok := parseMouseEvent(string(bytes), getFrameRows, logFunc); ok {
				select {
				case mouseChan <- ev:
				default:
					logFunc("Событие мыши пропущено: канал mouseChan переполнен")
				}
			}
			continue
		}

		// Обработка сложных клавиш (Escape-последовательности, например стрелочки)
		if bytes[0] == 27 && n > 1 {
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
				if n == 1 {
					keyName = "escape"
				} else {
					logFunc(fmt.Sprintf("Неизвестная escape-последовательность: %q", inputStr))
				}
			}

			if keyName != "" {
				sendKey(keyChan, KeyEvent{Key: keyName}, logFunc)
			}
			continue
		}

		// 3. Обработка сочетаний с Ctrl (в Raw Mode байты 1..26 соответствуют Ctrl+A..Ctrl+Z)
		if bytes[0] >= 1 && bytes[0] <= 26 && bytes[0] != 13 && bytes[0] != 9 {
			letter := string('a' + bytes[0] - 1)
			sendKey(keyChan, KeyEvent{Key: "ctrl+" + letter}, logFunc)
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
			rawKey := strings.ToLower(string(bytes))
			rusToEng := map[string]string{
				"й": "q", "ц": "w", "у": "e", "к": "r", "е": "t", "н": "y",
				"ф": "a", "ы": "s", "в": "d", "а": "f", "п": "g", "р": "h",
				"я": "z", "ч": "x", "с": "c", "м": "v", "и": "b", "т": "n",
			}

			if engKey, found := rusToEng[rawKey]; found {
				keyName = engKey
			} else {
				keyName = rawKey
			}
		}

		sendKey(keyChan, KeyEvent{Key: keyName}, logFunc)
	}
}

// sendKey безопасно отправляет событие клавиатуры в буферизированный канал
func sendKey(keyChan chan<- KeyEvent, ev KeyEvent, logFunc func(string)) {
	select {
	case keyChan <- ev:
	default:
		logFunc(fmt.Sprintf("Буфер канала клавиш переполнен, событие Key=%q пропущено", ev.Key))
	}
}

// parseMouseEvent декодирует строку формата SGR "\x1b[<button>;<x>;<y>M" (или m)
func parseMouseEvent(mouseStr string, getFrameRows func() uint, logFunc func(string)) (MouseEvent, bool) {
	str := mouseStr[3:]
	isDown := true
	if strings.HasSuffix(str, "m") {
		isDown = false
	}
	str = str[:len(str)-1]
	parts := strings.Split(str, ";")
	if len(parts) != 3 {
		logFunc(fmt.Sprintf("Ошибка парсинга события мыши '%s': ожидалось 3 части, получено %d", mouseStr, len(parts)))
		return MouseEvent{}, false
	}

	var btnRaw, termX, termY int
	_, err1 := fmt.Sscanf(parts[0], "%d", &btnRaw)
	_, err2 := fmt.Sscanf(parts[1], "%d", &termX)
	_, err3 := fmt.Sscanf(parts[2], "%d", &termY)
	if err1 != nil || err2 != nil || err3 != nil {
		logFunc(fmt.Sprintf("Ошибка парсинга координат мыши '%s': err1=%v, err2=%v, err3=%v", mouseStr, err1, err2, err3))
		return MouseEvent{}, false
	}

	xIdx := uint(termX - 1)
	yRow := uint(termY - 1)

	termHeight := getFrameRows()
	if yRow >= termHeight {
		logFunc(fmt.Sprintf("Событие мыши отклонено: Y-строка (%d) вне диапазона холста (%d)", yRow, termHeight))
		return MouseEvent{}, false
	}

	yIdx := (termHeight - 1 - yRow) * 2
	button := MouseButton(btnRaw & 67)
	if (btnRaw & 32) != 0 {
		button = MouseMove
	}

	return MouseEvent{
		X:      xIdx,
		Y:      yIdx,
		Button: button,
		IsDown: isDown,
	}, true
}
