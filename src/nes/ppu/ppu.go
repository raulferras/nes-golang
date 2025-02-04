package ppu

import (
	"github.com/raulferras/nes-golang/src/nes/gamePak"
	"github.com/raulferras/nes-golang/src/nes/types"
	"image"
	"image/color"
)

type PPU interface {
	WriteRegister(register types.Address, value byte)
	ReadRegister(register types.Address) byte
}

type Scanline uint16

type P2c02 struct {
	PpuControl Control
	PpuStatus  Status
	PpuMask    Mask // Controls the rendering of sprites and backgrounds
	vRam       LoopyRegister
	tRam       LoopyRegister
	fineX      uint8
	readBuffer byte
	oamAddr    byte

	cartridge    *gamePak.GamePak
	nameTables   [2 * NAMETABLE_SIZE]byte
	paletteTable [PALETTE_SIZE]byte
	// OAM (Object Attribute Memory) is internal memory inside the PPU.
	// Contains a display list of up to 64 sprites, where each sprite occupies 4 bytes
	oamData [OAMDATA_SIZE]byte

	// Background rendering
	bgNextTileId           byte
	bgNextAttribute        byte
	bgNextLowTile          byte
	bgNextHighTile         byte
	bgShifterTileLow       uint16
	bgShifterTileHigh      uint16
	bgShifterAttributeLow  uint16
	bgShifterAttributeHigh uint16

	// Sprite rendering
	oamDataScanline      [8]objectAttributeEntry
	spriteScanlineCount  byte
	spShifterPatternLow  [8]byte
	spShifterPatternHigh [8]byte

	cycle  uint32 // Current lifetime PPU Cycle. After warmup, ignored.
	warmup bool   // Indicates ppu is already warmed up (cycles went above 30000)

	renderCycle     uint16   // Current cycle inside a Scanline. From 0 to PPU_CYCLES_BY_SCANLINE
	currentScanline Scanline // Current vertical Scanline being rendered
	evenFrame       bool     // Is current Frame even?
	frame           uint16
	frameComplete   bool

	nmi              bool // NMI Interrupt thrown
	nameTableChanged bool

	// Render related
	renderByPixel   bool
	screen          *image.RGBA
	framePatternIDs [1024]byte // Screen representation with pattern ids and its position in screen. For debugging purposes.
	logger          *logger2c02
	debug           bool
}

func CreatePPU(cartridge *gamePak.GamePak, debug bool, logPath string) *P2c02 {
	debug = false
	ppu := &P2c02{
		cartridge:       cartridge,
		renderCycle:     0,
		currentScanline: 0,
		cycle:           0,
		vRam:            LoopyRegister{0, 0, 0, 0, 0, 0},
		tRam:            LoopyRegister{0, 0, 0, 0, 0, 0},
		fineX:           0,

		bgShifterTileLow:       0,
		bgShifterTileHigh:      0,
		bgShifterAttributeLow:  0,
		bgShifterAttributeHigh: 0,

		warmup:        false,
		renderByPixel: true,
		evenFrame:     true,
		screen:        image.NewRGBA(image.Rect(0, 0, types.SCREEN_WIDTH, types.SCREEN_HEIGHT)),

		logger: nil,
		debug:  debug,
	}

	if debug {
		ppu.logger = NewLogger2c02(debug, logPath)
	}

	return ppu
}

func (ppu *P2c02) Frame() *image.RGBA {
	return ppu.screen
}

func (ppu *P2c02) FramePattern() []byte {
	return ppu.nameTables[0:1024]
}

func (ppu *P2c02) VRam() LoopyRegister {
	return ppu.vRam
}
func (ppu *P2c02) TRam() LoopyRegister {
	return ppu.tRam
}
func (ppu *P2c02) FineX() uint8 {
	return ppu.fineX
}
func (ppu *P2c02) Scanline() Scanline {
	return ppu.currentScanline
}
func (ppu *P2c02) RenderCycle() uint16 {
	return ppu.renderCycle
}

func (ppu *P2c02) FrameNumber() uint16 {
	return ppu.frame
}

func (ppu *P2c02) Tick() {
	if ppu.debug && ppu.warmup == true {
		ppu.logger.log(ppu)
	}

	// VBlank logic
	if ppu.currentScanline == VBLANK_START_SCANLINE {
		if ppu.renderCycle == 1 {
			// TODO refactor to a method to set Vblank
			// TODO enabling VBlank only on ==1 and not on >=1 makes it difficult to start emulation inside a VBlank cycle. If changed, nmi triggering should be worked though.
			ppu.PpuStatus.VerticalBlankStarted = true

			if ppu.PpuControl.GenerateNMIAtVBlank {
				ppu.nmi = true
			}
		}
	} else if ppu.currentScanline == VBLANK_END_SCNALINE && ppu.renderCycle == 1 {
		ppu.PpuStatus.VerticalBlankStarted = false
		ppu.PpuStatus.Sprite0Hit = 0
	}

	// ------------------------------
	// Render logic
	ppu.renderLogic()

	//bit := ppu.registers.scrollX
	// Load new data into registers
	//if ppu.cycle%8 == 0 {
	//
	//}

	// Render logic end
	// ------------------------------

	// 341 PPU clock cycles have passed
	if ppu.renderCycle == PPU_CYCLES_BY_SCANLINE-1 {
		if ppu.currentScanline == 261 {
			ppu.evenFrame = !ppu.evenFrame
			ppu.currentScanline = 0
			ppu.frame++
			ppu.frameComplete = true
			//fmt.Printf("End of frame: %d\n", ppu.frame)
		} else {
			ppu.currentScanline++
		}
		ppu.renderCycle = 0
		if ppu.shouldSkipFirstCycleOnOddFrame() {
			ppu.renderCycle = 1
		}
	} else {
		ppu.renderCycle++
	}

	if ppu.cycle >= PPU_CYCLES_TO_WARMUP {
		ppu.warmup = true
	} else {
		ppu.cycle++
	}
}

func (ppu *P2c02) shouldSkipFirstCycleOnOddFrame() bool {
	return ppu.PpuMask.ShowBackground == 1 && ppu.scanlineIsVisibleOrIsPreRender() && ppu.evenFrame == false && ppu.currentScanline == 0 && ppu.renderCycle == 0
}

func (ppu *P2c02) incrementX() {
	if ppu.PpuMask.renderingEnabled() {
		if ppu.vRam.CoarseX() == 31 { // if CoarseX == 31
			ppu.vRam._coarseX = 0     // CoarseX = 0
			ppu.vRam._nameTableX ^= 1 // switch horizontal nametable
		} else {
			ppu.vRam._coarseX += 1 // CoarseX++
		}
	}
}

func (ppu *P2c02) incrementY() {
	if ppu.PpuMask.renderingEnabled() {
		if ppu.vRam.FineY() < 7 {
			ppu.vRam._fineY++
		} else {
			ppu.vRam.resetFineY()
			y := ppu.vRam.CoarseY()
			if y == 29 { // last row of tiles in a nametable
				ppu.vRam._coarseY = 0
				// Switch vertical NameTable
				ppu.vRam._nameTableY ^= 1
			} else if y == 31 {
				// pointer is in the attribute memory, we skip it
				ppu.vRam._coarseY = 0
			} else {
				ppu.vRam._coarseY++
			}
		}
	}
}

func (ppu *P2c02) transferX() {
	if ppu.PpuMask.renderingEnabled() {
		ppu.vRam._coarseX = ppu.tRam._coarseX
		ppu.vRam._nameTableX = ppu.tRam._nameTableX
	}
}

func (ppu *P2c02) transferY() {
	if ppu.PpuMask.renderingEnabled() {
		ppu.vRam._fineY = ppu.tRam._fineY
		ppu.vRam.setCoarseY(ppu.tRam.CoarseY())
		ppu.vRam.setNameTableY(ppu.tRam.NameTableY())
	}
}

func (ppu *P2c02) Nmi() bool {
	occurred := ppu.nmi
	ppu.nmi = false

	return occurred
}

func (ppu *P2c02) ResetNmi() {
	ppu.nmi = false
}

/*
	//$3F00 	    Universal background color
	//$3F01-$3F03 	Background palette 0
	//$3F05-$3F07 	Background palette 1
	//$3F09-$3F0B 	Background palette 2
	//$3F0D-$3F0F 	Background palette 3
	//$3F11-$3F13 	Sprite palette 0
	//$3F15-$3F17 	Sprite palette 1
	//$3F19-$3F1B 	Sprite palette 2
	//$3F1D-$3F1F 	Sprite palette 3
*/
func (ppu *P2c02) GetRGBColor(palette byte, colorIndex byte) color.RGBA {
	paletteColor := ppu.GetPaletteColor(palette, colorIndex)
	return color.RGBA{
		R: SystemPalette[paletteColor][0],
		G: SystemPalette[paletteColor][1],
		B: SystemPalette[paletteColor][2],
		A: 255,
	}
}

func (ppu *P2c02) GetPaletteColor(palette byte, colorIndex byte) byte {
	if palette > 0 && colorIndex == 0 {
		palette = 0
	}

	paletteAddress := types.Address((palette * 4) + colorIndex)
	value := ppu.Read(PaletteLowAddress + paletteAddress)
	return value
}

func (ppu *P2c02) Stop() {
	if ppu.debug {
		ppu.logger.Close()
	}
}

func (ppu *P2c02) Oam(index byte) []byte {

	return ppu.oamData[index : index+4]
}

func (ppu *P2c02) FrameComplete() bool {
	if ppu.frameComplete {
		ppu.frameComplete = false
		return true
	}

	return false
}
