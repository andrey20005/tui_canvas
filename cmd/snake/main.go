package main

import (
	"fmt"
	"time"

	"github.com/andrey20005/tui_canvas"
)

// ==========================================
// КОНСТАНТЫ
// ==========================================

var (
	ColorBg        = tui_canvas.NewColorRGB(5, 15, 5)
	ColorChess1    = tui_canvas.NewColorRGB(35, 110, 35)
	ColorChess2    = tui_canvas.NewColorRGB(40, 130, 40)
	ColorSnakeHead = tui_canvas.NewColorRGB(255, 140, 0)
	ColorSnakeTail = tui_canvas.NewColorRGB(128, 0, 128)
	ColorFruit     = tui_canvas.NewColorRGB(255, 0, 0)
	ColorTextWhite = tui_canvas.NewColorRGB(255, 255, 255)
)

const (
	FieldSize    = 20
	MoveInterval = 150 * time.Millisecond
)

// ==========================================
// ТИПЫ
// ==========================================

type GameState int

const (
	StateMenu GameState = iota
	StateGame
)

type Direction int

const (
	DirUp Direction = iota
	DirDown
	DirLeft
	DirRight
)

type Point struct{ X, Y int }

type Game struct {
	state     GameState
	score     int
	lastScore int
	focus     int // 0 = Play (выше), 1 = Quit (ниже)

	snake   []Point
	dir     Direction
	nextDir Direction
	fruit   Point

	cellSize uint
	fieldPxW uint
	fieldPxH uint
	offsetX  uint
	offsetY  uint

	playBtnBounds [4]int
	quitBtnBounds [4]int

	quitRequested bool
	accumulator   time.Duration
}

func NewGame() *Game {
	g := &Game{state: StateMenu}
	g.resetGame()
	return g
}

func (g *Game) resetGame() {
	g.snake = []Point{{10, 10}, {9, 10}, {8, 10}}
	g.dir = DirRight
	g.nextDir = DirRight
	g.score = 0
	g.spawnFruit()
}

func (g *Game) spawnFruit() {
	for {
		fx := int(time.Now().UnixNano()) % FieldSize
		fy := (int(time.Now().UnixNano()) / FieldSize) % FieldSize
		onSnake := false
		for _, p := range g.snake {
			if p.X == fx && p.Y == fy {
				onSnake = true
				break
			}
		}
		if !onSnake {
			g.fruit = Point{fx, fy}
			return
		}
		time.Sleep(1)
	}
}

// ==========================================
// ЛОГИКА
// ==========================================

func (g *Game) updateLogic() {
	if g.state != StateGame {
		return
	}
	g.dir = g.nextDir
	head := g.snake[0]

	// Y вверх: Up = Y++, Down = Y--
	switch g.dir {
	case DirUp:
		head.Y++
	case DirDown:
		head.Y--
	case DirLeft:
		head.X--
	case DirRight:
		head.X++
	}

	if head.X < 0 || head.X >= FieldSize || head.Y < 0 || head.Y >= FieldSize {
		g.gameOver()
		return
	}
	for _, p := range g.snake {
		if p.X == head.X && p.Y == head.Y {
			g.gameOver()
			return
		}
	}

	g.snake = append([]Point{head}, g.snake...)
	if head.X == g.fruit.X && head.Y == g.fruit.Y {
		g.score++
		g.spawnFruit()
	} else {
		g.snake = g.snake[:len(g.snake)-1]
	}
}

func (g *Game) gameOver() {
	g.lastScore = g.score
	g.state = StateMenu
	g.focus = 0
}

// ==========================================
// ВВОД
// ==========================================

func (g *Game) HandleKey(key string) {
	switch key {
	case "w", "up":
		if g.state == StateMenu {
			g.focus = 0
		} else if g.dir != DirDown {
			g.nextDir = DirUp
		}
	case "s", "down":
		if g.state == StateMenu {
			g.focus = 1
		} else if g.dir != DirUp {
			g.nextDir = DirDown
		}
	case "a", "left":
		if g.state == StateGame && g.dir != DirRight {
			g.nextDir = DirLeft
		}
	case "d", "right":
		if g.state == StateGame && g.dir != DirLeft {
			g.nextDir = DirRight
		}
	case "enter", "space":
		if g.state == StateMenu {
			if g.focus == 0 {
				g.resetGame()
				g.state = StateGame
			} else {
				g.quitRequested = true
			}
		}
	case "escape", "q", "ctrl+c", "й":
		g.quitRequested = true
	}
}

func (g *Game) HandleMouse(ev tui_canvas.MouseEvent) {
	if g.state == StateMenu && ev.Button == tui_canvas.MouseLeft && ev.IsDown {
		mx, my := int(ev.X), int(ev.Y)
		if inRect(mx, my, g.playBtnBounds) {
			g.resetGame()
			g.state = StateGame
		} else if inRect(mx, my, g.quitBtnBounds) {
			g.quitRequested = true
		}
	}
}

func inRect(x, y int, b [4]int) bool {
	return x >= b[0] && x < b[2] && y >= b[1] && y < b[3]
}

// ==========================================
// РЕНДЕРИНГ
// ==========================================

func (g *Game) calcGrid(cW, cH uint) {
	g.cellSize = (cH - 2) / FieldSize
	if (cW < cH) {
		g.cellSize = (cW - 2) / FieldSize
	}
	g.fieldPxW = uint(FieldSize) * g.cellSize
	g.fieldPxH = uint(FieldSize) * g.cellSize
	g.offsetX = (cW - g.fieldPxW) / 2
	g.offsetY = (cH - g.fieldPxH) / 2
}

// logicalToPixel: Y вверх везде, инверсии нет
func (g *Game) logicalToPixel(lx, ly int) (int, int) {
	return int(g.offsetX) + lx*int(g.cellSize),
		int(g.offsetY) + ly*int(g.cellSize)
}

func (g *Game) Render(canvas *tui_canvas.Canvas, text *tui_canvas.TextLayer) {
	cW, cH := canvas.Width(), canvas.Height()
	g.calcGrid(cW, cH)

	// Фон + шахматное поле
	canvas.FillShader(func(x, y int) tui_canvas.Color {
		if x >= int(g.offsetX) && x < int(g.offsetX+g.fieldPxW) &&
			y >= int(g.offsetY) && y < int(g.offsetY+g.fieldPxH) {
			lx := (x - int(g.offsetX)) / int(g.cellSize)
			ly := (y - int(g.offsetY)) / int(g.cellSize)
			if (lx+ly)%2 == 0 {
				return ColorChess1
			}
			return ColorChess2
		}
		return ColorBg
	})

	// Фрукт
	fx, fy := g.logicalToPixel(g.fruit.X, g.fruit.Y)
	// canvas.DrawEllipse(fx, fy, 4, 4, ColorFruit)
	// canvas.DrawEllipse(fx, fy, int(g.cellSize), int(g.cellSize), ColorFruit)
	canvas.FillEllipse(fx, fy, int(g.cellSize), int(g.cellSize), ColorFruit)
	// if (g.cellSize <= 3) {
	// 	canvas.FillRect(fx, fy, int(g.cellSize), int(g.cellSize), ColorFruit)
	// } else {
	// 	// canvas.DrawCircle(fx + (int(g.cellSize) - 1) / 2, fy + (int(g.cellSize) - 1) / 2, int(g.cellSize) / 2, ColorFruit)
	// 	canvas.FillCircle(fx + (int(g.cellSize) - 1) / 2, fy + (int(g.cellSize) - 1) / 2, int(g.cellSize) / 2, ColorFruit)
	// }

	// Змейка с градиентом
	for i, p := range g.snake {
		px, py := g.logicalToPixel(p.X, p.Y)
		t := 0.0
		if len(g.snake) > 1 {
			t = float64(i) / float64(len(g.snake)-1)
		}
		canvas.FillRect(px, py, int(g.cellSize), int(g.cellSize), ColorSnakeHead.Mix(ColorSnakeTail, t))
	}

	// Счёт над полем (больший Y = выше)
	if g.state == StateGame {
		s := fmt.Sprintf("Score: %d", g.score)
		// Верхний край поля в TextLayer: (offsetY + fieldPxH) / 2
		textY := int(g.offsetY+g.fieldPxH)/2 + 1
		if textY >= int(text.Height()) {
			textY = int(text.Height()) - 1
		}
		textX := (int(cW) - len(s)) / 2
		text.PrintAt(textX, textY, s, tui_canvas.TransparentTextShader{TextColor: ColorTextWhite})
	}

	if g.state == StateMenu {
		g.renderMenu(canvas, text, int(cW), int(cH))
	}
}

func (g *Game) renderMenu(canvas *tui_canvas.Canvas, text *tui_canvas.TextLayer, cW, cH int) {
	menuW, menuH := 24, 10
	// my — нижний край меню в TextLayer (Y вверх)
	mx := (int(text.Width()) - menuW) / 2
	my := (int(text.Height()) - menuH) / 2

	// Пиксельные координаты нижнего края меню
	pxX := mx
	pxY := my * 2 // 1 символ = 2 пикселя по Y

	// Полупрозрачный фон
	canvas.FillRectShader(pxX, pxY, menuW, menuH*2,
		func(absX, absY, relX, relY float64, idxX, idxY uint) (tui_canvas.Color, float64) {
			return tui_canvas.ColorBlack, 0.75
		})

	shader := tui_canvas.TransparentTextShader{TextColor: ColorTextWhite}

	scoreStr := fmt.Sprintf("Last Score: %d", g.lastScore)
	text.PrintAt(mx+(menuW-len(scoreStr))/2, my+6, scoreStr, shader)

	playStr := "  PLAY  "
	quitStr := "  QUIT  "
	if g.focus == 0 {
		playStr = "> PLAY <"
	} else {
		quitStr = "> QUIT <"
	}

	text.PrintAt(mx+(menuW-len(playStr))/2, my+4, playStr, shader)
	text.PrintAt(mx+(menuW-len(quitStr))/2, my+2, quitStr, shader)

	// Границы кнопок в пикселях Canvas (Y вверх)
	// PLAY: символ my+6..my+7 → пиксели (my+6)*2 .. (my+8)*2-1
	g.playBtnBounds = [4]int{pxX, pxY + 8, pxX + menuW, pxY + 12}
	// QUIT: символ my+4..my+5 → пиксели (my+4)*2 .. (my+6)*2-1
	g.quitBtnBounds = [4]int{pxX, pxY + 4, pxX + menuW, pxY + 8}
}

// ==========================================
// MAIN
// ==========================================

func main() {
	screen, err := tui_canvas.NewScreen("snake_debug.log")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}
	defer screen.Close()

	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	game := NewGame()
	lastTime := time.Now()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			dt := now.Sub(lastTime)
			lastTime = now

			game.accumulator += dt
			for game.accumulator >= MoveInterval {
				game.updateLogic()
				game.accumulator -= MoveInterval
			}

			screen.Draw(func(canvas *tui_canvas.Canvas, text *tui_canvas.TextLayer) {
				text.Clear()
				game.Render(canvas, text)
			})

			if game.quitRequested {
				return
			}

		case keyEv := <-screen.KeyEvents():
			game.HandleKey(keyEv.Key)
		case mouseEv := <-screen.MouseEvents():
			game.HandleMouse(mouseEv)
		case <-screen.ResizeEvents():
		}
	}
}