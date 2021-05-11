package nes

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// Helper methods
func Test_ppuctrlWriteFlag(t *testing.T) {
	ppu := aPPU()
	cases := []struct {
		name     string
		initial  byte
		write    byte
		expected byte
	}{
		{"writes on blank", 0x00, 1, 0b00000100},
		{"writes clear bits", 0xFF, 0, 0b11111011},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ppu.registers.ctrl = tt.initial
			ppu.ppuctrlWriteFlag(incrementMode, tt.write)
			assert.Equal(t, tt.expected, ppu.registers.ctrl)
		})
	}
}

func TestPPU_PPUCTRL_writes_are_ignored_first_30000_cycles(t *testing.T) {
	ppu := aPPU()
	for i := 0; i < 30000; i++ {
		ppu.Tick()
		ppu.WriteRegister(PPUCTRL, 0x11)

		if 0x11 == ppu.registers.ctrl {
			t.Error("writes to PPUCTRL should be ignored first 30000 cycles")
			t.FailNow()
		}
	}
}

func TestPPU_PPUCTRL_write(t *testing.T) {
	ppu := aPPU()

	for i := 0; i < 30001; i++ {
		ppu.Tick() // Advance ppu cycles
	}

	ppu.WriteRegister(PPUCTRL, 0xFF)

	assert.Equal(t, byte(0xFF), ppu.registers.ctrl)
}

func TestPPU_PPUMASK_write(t *testing.T) {
	ppu := aPPU()

	cases := []struct {
		name     string
		initial  byte
		write    byte
		expected byte
	}{
		{"writes on blank", 0x00, 0x10, 0x10},
		{"writes reset bits", 0xFF, 0x01, 0x01},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ppu.WriteRegister(PPUMASK, 0xFF)
			assert.Equal(t, byte(0xFF), ppu.registers.mask)
		})
	}
}

func TestPPU_PPUSTATUS_read(t *testing.T) {
	ppu := aPPU()
	ppu.registers.status = 0b11100000

	status := ppu.ReadRegister(PPUSTATUS)

	assert.Equal(t, byte(0b11100000), status)
}

func TestPPU_PPUSTATUS_reading_status_clears_bit7_and_the_address_latch(t *testing.T) {
	ppu := aPPU()
	ppu.registers.status = 0x80

	ppu.ReadRegister(PPUSTATUS)

	assert.Equal(t, byte(0), ppu.registers.status&0x80, "vblank flag should be cleared after reading PPUSTATUS")
	assert.Equal(t, byte(0), ppu.registers.addressLatch, "unexpected address latch")
}

// Reading PPUSTATUS within two cycles of the start of vertical blank will return 0 in bit 7 but clear the latch anyway, causing NMI to not occur that frame.
func TestPPUSTATUS_should_clear_latch_when_reading_within_two_cycles_of_sthe_start_of_vblank(t *testing.T) {
	t.Skipf("Waiting to implement VBlanks")
}

func TestPPUOAM_address_write(t *testing.T) {
	ppu := aPPU()

	ppu.WriteRegister(OAMADDR, 0xFF)

	assert.Equal(t, byte(0xFF), ppu.registers.oamAddr)
}

func TestPPUOAM_should_be_able_to_read(t *testing.T) {
	ppu := aPPU()
	ppu.oamData[0] = 0xFF

	value := ppu.ReadRegister(OAMDATA)

	assert.Equal(t, byte(0xFF), value)
}

func TestPPUOAM_should_be_able_to_write(t *testing.T) {
	ppu := aPPU()
	//ppu.oamData[0] = 0xFF

	ppu.WriteRegister(OAMDATA, 0xFF)

	assert.Equal(t, byte(0xFF), ppu.oamData[0])
	assert.Equal(t, byte(0x01), ppu.registers.oamAddr)
}

func TestPPUOAM_should_decay_if_not_refreshed_for_3000_cycles(t *testing.T) {
	t.Skip("should I really implement this?")
}

func TestPPUSCROLL_writes_twice(t *testing.T) {
	ppu := aPPU()
	scrollX := byte(0xFF)
	scrollY := byte(0xFF)
	ppu.WriteRegister(PPUSCROLL, scrollX)
	ppu.WriteRegister(PPUSCROLL, scrollY)

	assert.Equal(t, scrollX, ppu.registers.scrollX)
	assert.Equal(t, scrollY, ppu.registers.scrollY)
}

func TestPPU_PPUADDR_write_twice_to_set_address(t *testing.T) {
	cases := []struct {
		name     string
		hi       byte
		lo       byte
		expected Address
	}{
		{"writes address", 0x28, 0x10, 0x2810},
		{"writes address > 0x3FFF is mirrored down", 0x40, 0x20, 0x0020},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gamePak := CreateDummyGamePak()
			memory := CreatePPUMemory(gamePak)
			ppu := CreatePPU(memory)

			ppu.WriteRegister(PPUADDR, tt.hi)
			assert.Equal(t, Address(tt.hi)<<8, ppu.registers.ppuAddr)

			ppu.WriteRegister(PPUADDR, tt.lo)
			assert.Equal(t, tt.expected, ppu.registers.ppuAddr)
		})
	}
}

func TestPPU_PPUData_read(t *testing.T) {
	gamePak := CreateDummyGamePak()
	memory := CreatePPUMemory(gamePak)

	const PALETTE_VALUE = byte(0x20)
	const EXPECTED_VALUE = byte(0x15)

	cases := []struct {
		name          string
		addressToRead Address
		incrementMode byte
		firstRead     byte
		secondRead    byte
	}{
		{"buffered read, increment mode going across", 0x2600, 0, 0, EXPECTED_VALUE},
		{"buffered read, increment mode going down", 0x2600, 1, 0, EXPECTED_VALUE},
		{"reading from palette addresses does not buffer", 0x3F00, 0, PALETTE_VALUE, 0},
		{"reading from palette addresses does not buffer", 0x3FFF, 0, PALETTE_VALUE, PALETTE_VALUE},
	}

	ppu := CreatePPU(memory)
	ppu.Write(0x2600, EXPECTED_VALUE)
	ppu.Write(0x3F00, PALETTE_VALUE)
	ppu.Write(0x3FFF, PALETTE_VALUE)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ppu.registers.ppuAddr = tt.addressToRead
			ppu.ppuctrlWriteFlag(incrementMode, tt.incrementMode)
			expectedIncrement := Address(1)
			if tt.incrementMode == 1 {
				expectedIncrement = 32
			}

			// Dummy Read
			firstRead := ppu.ReadRegister(PPUDATA)
			assert.Equal(t, tt.firstRead, firstRead, "unexpected first read value")
			assert.Equal(t, (tt.addressToRead+expectedIncrement)&0x3FFF, ppu.registers.ppuAddr, "unexpected first read ppuAddr increment")

			secondRead := ppu.ReadRegister(PPUDATA)

			assert.Equal(t, tt.secondRead, secondRead, "unexpected second read value")

			assert.Equal(t, (tt.addressToRead+expectedIncrement*2)&0x3FFF, ppu.registers.ppuAddr, "unexpected second read ppuAddr increment")
		})
	}
}

func TestPPUDATA_is_instructed_to_read_address_and_mirrors(t *testing.T) {
	t.Skipf("Mirror still not implemented")
	gamePak := CreateDummyGamePak()
	memory := CreatePPUMemory(gamePak)
	ppu := CreatePPU(memory)

	ppu.WriteRegister(PPUADDR, 0x3F)
	ppu.WriteRegister(PPUADDR, 0xFF)

	// Dummy Read
	ppu.ReadRegister(PPUDATA)
	assert.Equal(t, Address(0x0000), ppu.registers.ppuAddr, "ppuAddr(cpu@0x2006) must increment on each read to cpu@0x2007")
}

func TestPPU_PPUData_write(t *testing.T) {
	gamePak := CreateDummyGamePak()
	memory := CreatePPUMemory(gamePak)

	const PALETTE_VALUE = byte(0x20)
	const EXPECTED_VALUE = byte(0x15)

	cases := []struct {
		name           string
		addressToWrite Address
		incrementMode  byte
		valueToWrite   byte
	}{
		{"write, increment mode going across", 0x2600, 0, EXPECTED_VALUE},
		{"write, increment mode going down", 0x2600, 1, EXPECTED_VALUE},
		{"write into palette, low edge", 0x3F00, 0, PALETTE_VALUE},
		{"write into palette, high edge", 0x3FFF, 0, PALETTE_VALUE},
	}

	ppu := CreatePPU(memory)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ppu.registers.ppuAddr = tt.addressToWrite
			ppu.ppuctrlWriteFlag(incrementMode, tt.incrementMode)
			expectedIncrement := Address(1)
			if tt.incrementMode == 1 {
				expectedIncrement = 32
			}

			ppu.WriteRegister(PPUDATA, tt.valueToWrite)

			writtenValue := ppu.Read(tt.addressToWrite)
			assert.Equal(t, tt.valueToWrite, writtenValue, "unexpected value written")
			assert.Equal(t, (tt.addressToWrite+expectedIncrement)&0x3FFF, ppu.registers.ppuAddr, "unexpected first read ppuAddr increment")
		})
	}
}

func TestOAMDMA(t *testing.T) {

}
