package tuicanvas

// KeyEvent описывает событие нажатия клавиши или их сочетания на клавиатуре.
type KeyEvent struct {
	// Key содержит строковое представление клавиши в нижнем регистре.
	// Обычные: "w", "a", "space", "enter", "escape".
	// Стрелочки: "up", "down", "left", "right".
	// Модификаторы: "ctrl+c", "ctrl+z", "shift+up", "shift+down".
	Key string
}

// MouseButton представляет собой тип для идентификации кнопок мыши.
type MouseButton int

const (
	MouseLeft       MouseButton = 0
	MouseMiddle     MouseButton = 1
	MouseRight      MouseButton = 2
	MouseScrollUp   MouseButton = 64
	MouseScrollDown MouseButton = 65
	MouseMove       MouseButton = 99 // простое движение без клика
)

// MouseEvent описывает действия с мышью (клики, перемещения, скролл).
type MouseEvent struct {
	// Целочисленные координаты пикселя на холсте Canvas (X: 0..width, Y: 0..height)
	// Ось Y смотрит ВВЕРХ.
	X uint
	Y uint

	// Какая кнопка мыши совершила действие
	Button MouseButton

	// true — кнопка нажата, false — кнопка отпущена (или мышь просто двигалась)
	IsDown bool
}

// ResizeEvent описывает изменение размеров окна терминала.
type ResizeEvent struct {
	Width  uint
	Height uint
}
