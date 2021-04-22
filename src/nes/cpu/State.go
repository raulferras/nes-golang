package cpu

import (
	"fmt"
	"github.com/raulferras/nes-golang/src/nes/defs"
	"github.com/raulferras/nes-golang/src/utils"
	"regexp"
	"strconv"
	"strings"
)

type State struct {
	Registers          Cpu6502Registers
	CurrentInstruction defs.Instruction
	RawOpcode          []byte
	EvaluatedAddress   defs.Address
	CyclesSinceReset   uint16
}

func CreateState(cpu Cpu6502) State {
	pc := cpu.Registers().Pc

	var rawOpcode []byte
	rawOpcode = append(rawOpcode, cpu.Read(pc))
	pc++
	instruction := cpu.instructions[rawOpcode[0]]

	for i := byte(0); i < (instruction.Size() - 1); i++ {
		rawOpcode = append(rawOpcode, cpu.Read(pc+defs.Address(i)))
	}

	_, evaluatedAddress, _, _ := cpu.addressEvaluators[instruction.AddressMode()](pc)

	state := State{
		*cpu.Registers(),
		instruction,
		rawOpcode,
		evaluatedAddress,
		cpu.Cycle,
	}

	return state
}

func CreateStateFromNesTestLine(nesTestLine string) State {
	fields := strings.Fields(nesTestLine)
	_ = fields

	blocks := utils.StringSplitByRegex(nesTestLine)

	result := utils.HexStringToByteArray(blocks[0])
	pc := defs.CreateAddress(result[1], result[0])

	fields = strings.Fields(blocks[1])
	opcode := utils.HexStringToByteArray(fields[0])

	flagFields := strings.Fields(blocks[3])

	r, _ := regexp.Compile("CYC:([0-9]+)$")
	cpuCyclesString := r.FindStringSubmatch(nesTestLine)

	cpuCycles, _ := strconv.ParseUint(cpuCyclesString[1], 10, 16)

	state := State{
		Cpu6502Registers{
			utils.NestestDecodeRegisterFlag(flagFields[0]),
			utils.NestestDecodeRegisterFlag(flagFields[1]),
			utils.NestestDecodeRegisterFlag(flagFields[2]),
			pc,
			utils.NestestDecodeRegisterFlag(flagFields[4]),
			utils.NestestDecodeRegisterFlag(flagFields[3]),
		},
		defs.CreateInstruction(
			strings.Fields(blocks[2])[0],
			defs.Implicit,
			nil,
			0,
			0,
		),
		opcode,
		defs.CreateAddress(0x00, 0x00),
		uint16(cpuCycles),
	}

	return state
}

func (state State) Equals(b State) bool {
	if state.Registers.Pc != b.Registers.Pc ||
		state.Registers.A != b.Registers.A ||
		state.Registers.X != b.Registers.X ||
		state.Registers.Y != b.Registers.Y ||
		state.Registers.Sp != b.Registers.Sp ||
		state.Registers.statusRegister() != b.Registers.statusRegister() ||
		state.CyclesSinceReset != b.CyclesSinceReset {
		return false
	}

	return true
}

func (state State) ToString() string {
	msg := fmt.Sprintf("%X ", state.Registers.Pc)
	msg += fmt.Sprintf("%X ", state.RawOpcode)
	msg += fmt.Sprintf("%s - ", state.CurrentInstruction.Name())
	msg += fmt.Sprintf(
		"A:%X X:%X Y:%X P:%X SP:%X ",
		state.Registers.A,
		state.Registers.X,
		state.Registers.Y,
		state.Registers.statusRegister(),
		state.Registers.Sp,
	)
	// Cycles
	msg += fmt.Sprintf("PPU:%d,%d CYC:%d", 0, 0, state.CyclesSinceReset)

	return msg
}
