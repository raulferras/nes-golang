package nes

import (
	"strings"
)

func myHex(n uint16, d int) string {
	s := strings.Repeat("0", d)
	i := d - 1
	for i >= 0 {
		c := "0123456789ABCDEF"[n&0xF]
		s = s[:i] + string(c) + s[i+1:]
		i--
		n >>= 4
	}

	return s
}

func (cpu *Cpu6502) Disassemble(start Address, end Address) map[Address]string {
	disassembledCode := make(map[Address]string)
	addr := uint16(start)
	value := byte(0x00)
	lo := byte(0x00)
	hi := byte(0x00)

	for addr <= uint16(end) {
		lineAddr := addr

		// Prefix line with instruction address
		sInst := "$" + myHex(addr, 4) + ": "

		// Read instruction, and get its readable name
		opcode := cpu.memory.Peek(Address(addr))
		addr++
		instruction := cpu.instructions[opcode]

		if len(instruction.Name()) == 0 {
			sInst += "0x" + myHex(uint16(opcode), 2) + "? "
		} else {
			sInst += instruction.Name() + " "
		}

		if instruction.AddressMode() == Implicit {
			sInst += " {IMP}"
		} else if instruction.AddressMode() == Immediate {
			value = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "#$" + myHex(uint16(value), 2) + " {IMM}"
		} else if instruction.AddressMode() == ZeroPage {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = 0x00
			sInst += "$" + myHex(uint16(lo), 2) + " {ZP0}"
		} else if instruction.AddressMode() == ZeroPageX {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = 0x00
			sInst += "$" + myHex(uint16(lo), 2) + ", X {ZPX}"
		} else if instruction.AddressMode() == ZeroPageY {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = 0x00
			sInst += "$" + myHex(uint16(lo), 2) + ", Y {ZPY}"
		} else if instruction.AddressMode() == IndirectX {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = 0x00
			sInst += "($" + myHex(uint16(lo), 2) + ", X) {IZX}"
		} else if instruction.AddressMode() == IndirectY {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = 0x00
			sInst += "($" + myHex(uint16(lo), 2) + "), Y {IZY}"
		} else if instruction.AddressMode() == Absolute {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "$" + myHex(uint16(CreateWord(lo, hi)), 4) + " {ABS}"
		} else if instruction.AddressMode() == AbsoluteXIndexed {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "$" + myHex(uint16(CreateWord(lo, hi)), 4) + ", X {ABX}"
		} else if instruction.AddressMode() == AbsoluteYIndexed {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "$" + myHex(uint16(CreateWord(lo, hi)), 4) + ", Y {ABY}"
		} else if instruction.AddressMode() == Indirect {
			lo = cpu.memory.Peek(Address(addr))
			addr++
			hi = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "($" + myHex(uint16(CreateWord(lo, hi)), 4) + ") {IND}"
		} else if instruction.AddressMode() == Relative {
			value = cpu.memory.Peek(Address(addr))
			addr++
			sInst += "$" + myHex(uint16(value), 2) + " [$" + myHex(addr+uint16(value), 4) + "] {REL}"
		}

		disassembledCode[Address(lineAddr)] = sInst
	}

	return disassembledCode
}
