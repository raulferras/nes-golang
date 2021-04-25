package main

import (
	"fmt"
	r "github.com/lachee/raylib-goplus/raylib"
	"github.com/raulferras/nes-golang/src/graphics"
	"github.com/raulferras/nes-golang/src/nes"
	"math/rand"
	"time"
)

var font *r.Font

func main() {
	rand.Seed(time.Now().UnixNano())

	// Init Window System
	r.InitWindow(800, 800, "NES golang")
	r.SetTraceLogLevel(r.LogWarning)
	r.SetTargetFPS(60)
	font = r.LoadFont("./assets/Pixel_NES.otf")

	graphics.InitDrawer()

	fmt.Printf("Nes Emulator\n")
	path := "./roms/nestest/nestest.nes"
	//path :="./roms/Donkey Kong (World) (Rev A).nes"
	//path := "./roms/Super Mario Bros. (World).nes"
	//path = "./roms/Mega Man 2 (Europe).nes"
	gamePak := nes.CreateGamePakFromROMFile(path)

	printRomInfo(&gamePak)

	console := nes.CreateNes(&gamePak, nes.NesDebugger{})
	//console.InsertGamePak(&gamePak)

	//
	console.Start()

	for !r.WindowShouldClose() {
		// Update emulator

		// Draw
		draw(console)
	}
	r.CloseWindow()
}

func printRomInfo(gamePak *nes.GamePak) {
	inesHeader := gamePak.Header()

	if inesHeader.HasTrainer() {
		fmt.Println("Rom has trainer")
	} else {
		fmt.Println("Rom has no trainer")
	}

	fmt.Println("PRG:", inesHeader.ProgramSize(), "x 16KB Banks")
	fmt.Println("CHR:", inesHeader.CHRSize(), "x 8KB Banks")
	fmt.Println("Mapper:", inesHeader.MapperNumber())
	fmt.Println("Tv System:", inesHeader.TvSystem())
}

func draw(console nes.Nes) {
	r.BeginDrawing()
	r.ClearBackground(r.Black)

	drawEmulation()
	drawDebug(console)

	r.EndDrawing()
}

func drawEmulation() {
	// TODO
}

const DEBUG_X_OFFSET = 380

func drawDebug(console nes.Nes) {
	x := DEBUG_X_OFFSET
	y := 10
	r.DrawFPS(0, 0)

	textColor := r.RayWhite

	r.SetTextureFilter(font.Texture, r.FilterPoint)

	// Status Register
	graphics.DrawText("STATUS:", x, y, textColor)
	graphics.DrawText("N", x+70, y, textColor)

	// Program counter
	msg := fmt.Sprintf("PC: 0x%X", console.Debugger().ProgramCounter())
	graphics.DrawText(msg, x, y+15, textColor)

	//registers := fmt.Sprintf("A:0x%0X X:0x%X Y:0x%X P:0x%X SP:0x%X", 0, 0, 0, 0, 0)
	//position := r.Vector2{X: 380, Y: 20}
	//r.DrawTextEx(*font, registers, position, fontSize, 0, r.RayWhite)

	drawASM(console)
	drawCHR(console, font)
}

func drawASM(console nes.Nes) {
	textColor := r.RayWhite
	yOffset := 40
	yIteration := 0
	ySeparation := 15
	disassembled := console.Debugger().Disassembled()

	for i := 0; i < 20; i++ {
		currentAddress := console.Debugger().ProgramCounter() - 10 + nes.Address(i)
		if currentAddress == console.Debugger().ProgramCounter() {
			textColor = r.GopherBlue
		} else {
			textColor = r.White
		}

		code := disassembled[currentAddress]
		if len(code) > 0 {
			graphics.DrawText(code, 380, yOffset+(yIteration*ySeparation), textColor)
			yIteration++
		}
	}
}

func drawCHR(console nes.Nes, font *r.Font) {
	x := DEBUG_X_OFFSET
	y := 40 + 15*20 + 20
	decodedPatternTable := console.Debugger().PatternTable(0)

	// CHR Left Container
	r.DrawRectangle(x-5, y-5, 16*8+10, 16*8+10, r.RayWhite)
	posX := 0
	posY := 0
	scale := 3

	for i := 0; i < 128*128; i++ {
		pixelValue := decodedPatternTable[i]

		color := pixelValueToColor(pixelValue.Color)
		posX = pixelValue.X + (pixelValue.X * (scale - 1)) + DEBUG_X_OFFSET
		posY = pixelValue.Y + (pixelValue.Y * (scale - 1)) + y
		r.DrawRectangle(posX, posY, scale, scale, color)
		//r.DrawPixel(posX, posY, color)
		//r.DrawPixel(posX, posY+1, color)
		//r.DrawPixel(posX+1, posY, color)
		//r.DrawPixel(posX+1, posY+1, color)

		/*color := pixelValueToColor(pixelValue)
		posX = (i%128)*2 + x
		posY = int((i*2)/128) + y

		r.DrawPixel(posX, posY, color)
		r.DrawPixel(posX, posY+1, color)
		r.DrawPixel(posX+1, posY, color)
		r.DrawPixel(posX+1, posY+1, color)
		*/
	}
	//fmt.Printf("%d - %d\n", posX, posY)
	// CHR Right Container
	//r.DrawRectangle(x, y, 16*8, 16*8, r.RayWhite)
}

func pixelValueToColor(pixelValue []byte) r.Color {
	if len(pixelValue) != 3 {
		return r.Red
	}

	//min := 0
	//max := 254
	return r.NewColor(
		pixelValue[0],
		pixelValue[1],
		pixelValue[2],
		//uint8(rand.Intn(max-min+1)+min),
		//uint8(rand.Intn(max-min+1)+min),
		//uint8(rand.Intn(max-min+1)+min),
		255,
	)
}
