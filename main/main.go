package main

import (
	"fmt"
	"time"
	"github.com/andrey20005/tui_canvas"
)

func main() {
	// Создаем экран и включаем опциональное логирование в файл "debug.log"
	// Если передать "", логирования не будет
	screen, err := tuicanvas.NewScreen("debug.log")
	if err != nil {
		fmt.Println("Ошибка инициализации экрана:", err)
		return
	}
	// Гарантируем, что терминал восстановится в любом случае
	defer screen.Close()

	// Создаем таймер на 30 FPS (~33мс на кадр)
	ticker := time.NewTicker(time.Second / 30)
	defer ticker.Stop()

	// Координаты нашей рисуемой точки (для управления с WASD)
	playerX, playerY := 0.0, 0.0

	for {
		select {
		case <-ticker.C:
			// НАСТУПИЛ НОВЫЙ КАДР: РЕНДЕРИНГ
			canvas := screen.Canvas()

			// Заливаем фон красивым процедурным шейдером (синий градиент)
			canvas.FillShaderCoords(func(x, y float64) tuicanvas.Color {
				return tuicanvas.NewColorFloat(0.0, 0.0, (y+1.0)*0.3)
			})

			// Поверх шейдера рисуем кружок нашего "игрока" с помощью FillCoordsAlpha
			canvas.FillShaderCoordsAlpha(func(x, y float64) (tuicanvas.Color, float64) {
				// Считаем расстояние от текущих координат игрока
				dx := x - playerX
				dy := y - playerY
				if dx*dx+dy*dy < 0.02 { // Маленький радиус
					return tuicanvas.ColorYellow, 1.0
				}
				return tuicanvas.ColorBlack, 0.0
			})

			// Выводим холст на экран
			screen.Update()

		case keyEv := <-screen.KeyEvents():
			// ОБРАБОТКА КЛАВИАТУРЫ
			switch keyEv.Key {
			case "escape", "q", "ctrl+c", "й":
				// Безопасный выход из игры
				return
			case "w", "up":
				playerY += 0.1
			case "s", "down":
				playerY -= 0.1
			case "a", "left":
				playerX -= 0.1
			case "d", "right":
				playerX += 0.1
			}

		case mouseEv := <-screen.MouseEvents():
			// ОБРАБОТКА МЫШИ
			// Если зажата левая кнопка мыши (MouseLeft), телепортируем желтый кружок туда
			if mouseEv.Button == tuicanvas.MouseLeft && mouseEv.IsDown {
				playerX = mouseEv.FX
				playerY = mouseEv.FY
			}

		case <-screen.ResizeEvents():
			// ОБРАБОТКА РЕСАЙЗА
			// Сам экран Screen уже перестроил Canvas внутри watchResize,
			// нам здесь ничего делать не нужно, в следующем тике кадра 
			// графика автоматически отрисуется под новый размер.
		}
	}
}


// package main

// import (
// 	"fmt"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"golang.org/x/term"
// )

// func main() {

// 	if !term.IsTerminal(int(os.Stdout.Fd())) {
// 		fmt.Println("Вывод идет не в терминал!")
// 		return
// 	}

// 	// Получаем размер: width (колонки) и height (строки)
// 	width, height, err := term.GetSize(int(os.Stdout.Fd()))
// 	if err != nil {
// 		fmt.Println("Ошибка получения размера:", err)
// 		return
// 	}

// 	fmt.Printf("Текущий размер терминала: Ширина = %d колонок, Высота = %d строк\n", width, height)

// 	// 1. Создаем канал для сигналов. ОС будет присылать туда данные типа os.Signal
// 	sigChan := make(chan os.Signal, 1)

// 	// 2. Регистрируем канал в системном диспетчере.
// 	// Говорим: "Если придет SIGWINCH (ресайз) или Interrupt (Ctrl+C) — отправь их в sigChan"
// 	signal.Notify(sigChan, syscall.SIGWINCH, os.Interrupt)

// 	fmt.Println("Программа запущена. Попробуй изменить размер окна терминала или нажать Ctrl+C...")

// 	// 3. Запускаем бесконечный цикл обработки
// 	for {
// 		// Ждем сигнал из канала (поток блокируется и спит, пока ОС молчит)
// 		sig := <-sigChan

// 		switch sig {
// 		case syscall.SIGWINCH:
// 			fmt.Println(" [Сигнал] Ого! Ты изменил размер окна!")
// 			// Тут в реальном TUI мы будем вызывать получение нового размера и делать canvas.Resize()
// 			// Получаем размер: width (колонки) и height (строки)
// 			width, height, err := term.GetSize(int(os.Stdout.Fd()))
// 			if err != nil {
// 				fmt.Println("Ошибка получения размера:", err)
// 				return
// 			}

// 			fmt.Printf("Текущий размер терминала: Ширина = %d колонок, Высота = %d строк\n", width, height)

// 		case os.Interrupt:
// 			fmt.Println("\n [Сигнал] Нажат Ctrl+C. Безопасно завершаем работу...")
// 			return // Выходим из программы
// 		}
// 	}
// }



// import (
// 	"fmt"
// 	"math"

// 	tuicanvas "github.com/andrey20005/tui_canvas"
// )

// func main() {
// 	// 1. Создаем начальный холст (40x20 пикселей)
// 	// В терминале это займет 40 колонок в ширину и 10 строк в высоту
// 	fmt.Println("=== 1. Создание холста 40x20 и заливка градиентом ===")
// 	canvas := tuicanvas.NewCanvas(40, 20)

// 	// Заливаем радиальным градиентом от центра (0,0)
// 	canvas.FillShaderCoords(func(x, y float64) tuicanvas.Color {
// 		// Считаем расстояние от центра
// 		dist := math.Sqrt(x*x + y*y)

// 		// Создаем плавное затухание от центра к краям
// 		// Центр будет ярко-зеленым/синим (циановым), края — темными
// 		r := 0.0
// 		g := 1.0 - dist
// 		b := 0.8 - dist*0.5

// 		return tuicanvas.NewColorFloat(r, g, b)
// 	})

// 	canvas.Render()

// 	// 2. Увеличиваем размер (60x30 пикселей)
// 	// Картинка должна остаться в центре, вокруг появятся черные поля
// 	fmt.Println("\n=== 2. Увеличение размера до 60x30 (центрирование и черные поля) ===")
// 	canvas.Resize(60, 30)
// 	canvas.Render()

// 	// 3. Уменьшаем размер (30x16 пикселей)
// 	// Картинка должна симметрично обрезаться по краям
// 	fmt.Println("\n=== 3. Уменьшение размера до 30x16 (центрированное обрезание) ===")
// 	canvas.Resize(30, 16)
// 	canvas.Render()
// }
