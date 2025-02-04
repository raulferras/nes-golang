package nes

import (
	"github.com/raulferras/nes-golang/src/nes/cpu"
	"github.com/raulferras/nes-golang/src/nes/types"
)

// Non Maskable Interrupt
func (cpu6502 *Cpu6502) nmi() {

	cpu6502.pushStack(byte(cpu6502.registers.Pc >> 8))
	cpu6502.pushStack(byte(cpu6502.registers.Pc))
	cpu6502.pushStack(cpu6502.registers.Status)

	cpu6502.registers.Pc = cpu6502.read16(0xFFFA)
}

func (cpu6502 *Cpu6502) irq() {

	cpu6502.pushStack(byte(cpu6502.registers.Pc >> 8))
	cpu6502.pushStack(byte(cpu6502.registers.Pc))
	cpu6502.pushStack(cpu6502.registers.Status)

	cpu6502.registers.Pc = cpu6502.read16(0xFFFE)
}

func (cpu6502 *Cpu6502) evalImplicit(programCounter types.Address) (finalAddress types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	finalAddress = 0
	cycles = 0
	return
}

/**
 * Immediate addressing allows the programmer to directly specify an 8 bit constant within the instruction.
 * It is indicated by a '#' symbol followed by an numeric expression.
 */
func (cpu6502 *Cpu6502) evalImmediate(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	address = programCounter
	cycles = 0
	opcodeOperand = [3]byte{byte(programCounter)}
	pageCrossed = false
	return
}

func (cpu6502 *Cpu6502) evalZeroPage(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	// 2 bytes
	var low = cpu6502.memory.Read(programCounter)

	address = types.Address(low)
	opcodeOperand = [3]byte{low}

	return
}

func (cpu6502 *Cpu6502) evalZeroPageX(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	registers := cpu6502.registers
	var low = cpu6502.memory.Read(programCounter) + registers.X

	address = types.Address(low) & 0xFF
	opcodeOperand = [3]byte{low}

	return
}

func (cpu6502 *Cpu6502) evalZeroPageY(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	registers := cpu6502.registers
	var low = cpu6502.memory.Read(programCounter) + registers.Y

	address = types.Address(low) & 0xFF
	opcodeOperand = [3]byte{low}
	return
}

func (cpu6502 *Cpu6502) evalAbsolute(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	low := cpu6502.memory.Read(programCounter)
	programCounter += 1

	// Bug: Missing incrementing programCounter
	high := cpu6502.memory.Read(programCounter)

	address = types.CreateAddress(low, high)
	opcodeOperand = [3]byte{low, high}

	return
}

func (cpu6502 *Cpu6502) evalAbsoluteXIndexed(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	low := cpu6502.memory.Read(programCounter)

	high := cpu6502.memory.Read(programCounter + 1)

	address = types.CreateAddress(low, high)
	address += types.Address(cpu6502.registers.X)

	opcodeOperand = [3]byte{low, high}
	pageCrossed = memoryPageDiffer(address-types.Address(cpu6502.registers.X), address)

	return
}

func (cpu6502 *Cpu6502) evalAbsoluteYIndexed(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	low := cpu6502.memory.Read(programCounter)
	high := cpu6502.memory.Read(programCounter + 1)

	address = types.CreateAddress(low, high)
	address += types.Address(cpu6502.registers.Y)

	pageCrossed = memoryPageDiffer(address-types.Address(cpu6502.registers.Y), address)
	opcodeOperand = [3]byte{low, high}

	return
}

// Address Mode: Indirect
// The supplied 16-bit address is read to get the actual 16-bit address.
// This is Instruction is unusual in that it has a bug in the hardware! To emulate its
// function accurately, we also need to emulate this bug. If the low byte of the
// supplied address is 0xFF, then to read the high byte of the actual address
// we need to cross a page boundary. This doesnt actually work on the chip as
// designed, instead it wraps back around in the same page, yielding an
// invalid actual address
// Example: supplied address is (0x1FF), LSB will be 0x00 and MSB will be 0x01 instead of 0x02.

// If the 16-bit argument of an Indirect JMP is located between 2 pages (0x01FF and 0x0200 for example),
// then the LSB will be read from 0x01FF and the MSB will be read from 0x0100.
// This is an actual hardware bug in early revisions of the 6502 which happen to be present
// in the 2A03 used by the NES.
func (cpu6502 *Cpu6502) evalIndirect(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	// Get Pointer types.Address
	ptrLow := cpu6502.memory.Read(programCounter)
	ptrHigh := cpu6502.memory.Read(programCounter + 1)

	ptrAddress := types.CreateAddress(ptrLow, ptrHigh)
	address = cpu6502.read16Bugged(ptrAddress)
	opcodeOperand = [3]byte{ptrLow, ptrHigh}

	return
}

func (cpu6502 *Cpu6502) evalIndirectX(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	operand := cpu6502.memory.Read(programCounter)
	opcodeOperand = [3]byte{operand}

	operand += cpu6502.registers.X
	operand &= 0xFF

	effectiveLow := cpu6502.memory.Read(types.Address(operand))
	effectiveHigh := cpu6502.memory.Read(types.Address(operand + 1)) // automatic warp around

	address = types.CreateAddress(effectiveLow, effectiveHigh)

	return
}

func (cpu6502 *Cpu6502) evalIndirectY(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	operand := cpu6502.memory.Read(programCounter)

	lo := cpu6502.memory.Read(types.Address(operand))
	hi := cpu6502.memory.Read(types.Address(operand + 1)) // automatic warp around

	address = types.CreateAddress(lo, hi)
	address += types.Word(cpu6502.registers.Y)

	pageCrossed = address&0xFF00 != types.Address(hi)<<8
	opcodeOperand = [3]byte{lo, hi}
	return
}

func (cpu6502 *Cpu6502) evalRelative(programCounter types.Address) (address types.Address, opcodeOperand [3]byte, cycles int, pageCrossed bool) {
	operand := cpu6502.memory.Read(programCounter)

	address = programCounter + 1
	if operand < 0x80 {
		address += types.Address(operand)
	} else {
		address += types.Address(operand) - 0x100
	}

	opcodeOperand = [3]byte{operand}

	return
}

/*
	ADC  Add Memory to Accumulator with Carry
     A + M + C -> A, C                N Z C I D V
                                      + + + - - +

     addressing    assembler    opc  bytes  cycles
     --------------------------------------------
     immidiate     ADC #oper     69    2     2
     zeropage      ADC oper      65    2     3
     zeropage,X    ADC oper,X    75    2     4
     Absolute      ADC oper      6D    3     4
     Absolute,X    ADC oper,X    7D    3     4*
     Absolute,Y    ADC oper,Y    79    3     4*
     (Indirect,X)  ADC (oper,X)  61    2     6
	 (Indirect),Y  ADC (oper),Y  71    2     5*

	http://www.righto.com/2012/12/the-6502-overflow-flag-explained.html
	https://forums.nesdev.com/viewtopic.php?t=6331
*/
func (cpu6502 *Cpu6502) adc(info cpu.OperationMethodArgument) bool {
	carryIn := cpu6502.registers.CarryFlag()
	a := cpu6502.registers.A
	value := cpu6502.memory.Read(info.OperandAddress)
	adc := uint16(a) + uint16(value) + uint16(carryIn)
	adc8 := cpu6502.registers.A + value + cpu6502.registers.CarryFlag()

	cpu6502.registers.A = adc8
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.SetCarryFlag(adc > 0xFF)

	// The exclusive-or bitwise operator is a neat little tool to check if the sign of two numbers is the same
	// If the sign of the sum matches either the sign of A or the sign of v, then you don't overflow
	if ((uint16(a) ^ adc) & (uint16(value) ^ adc) & 0x80) > 0 {
		cpu6502.registers.SetOverflowFlag(true)
	} else {
		cpu6502.registers.SetOverflowFlag(false)
	}

	return true
}

//	Performs a logical AND on the operand and the Accumulator and stores the result in the Accumulator
//
// 	Addressing Mode 	Assembly Language Form 	Opcode 	# Bytes 	# Cycles
// 	Immediate 			AND #Operand 			29 		2 			2
//	Zero Page 			AND Operand 			25 		2 			3
//	Zero Page, X 		AND Operand, X 			35 		2 			4
//	Absolute 			AND Operand 			2D 		3 			4
//	Absolute, X 		AND Operand, X 			3D 		3 			4*
//	Absolute, Y 		AND Operand, Y 			39 		3 			4*
//	(Indirect, X) 		AND (Operand, X)	 	21 		2 			6
//	(Indirect), Y 		AND (Operand), Y 		31 		2 			5*
//	* Add 1 if page boundary is crossed.
func (cpu6502 *Cpu6502) and(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.A &= cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)

	return true
}

/*
	ASL  Shift Left One Bit (Memory or Accumulator)

     C <- [76543210] <- 0             N Z C I D V
                                      + + + - - -

     addressing    assembler    opc  bytes  cycles
     --------------------------------------------
     Accumulator   ASL A         0A    1     2
     zeropage      ASL oper      06    2     5
     zeropage,X    ASL oper,X    16    2     6
	 Absolute      ASL oper      0E    3     6
*/
func (cpu6502 *Cpu6502) asl(info cpu.OperationMethodArgument) bool {
	if info.AddressMode == cpu.Implicit {
		cpu6502.registers.SetCarryFlag(cpu6502.registers.A>>7&0x01 == 1)
		cpu6502.registers.A = cpu6502.registers.A << 1
		cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)
		cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	} else {
		value := cpu6502.memory.Read(info.OperandAddress)
		cpu6502.registers.SetCarryFlag(value>>7&0x01 == 1)
		value = value << 1
		cpu6502.memory.Write(info.OperandAddress, value)
		cpu6502.registers.UpdateNegativeFlag(value)
		cpu6502.registers.UpdateZeroFlag(value)
	}

	return false
}

func (cpu6502 *Cpu6502) addBranchCycles(info cpu.OperationMethodArgument) bool {
	cpu6502.opCyclesLeft++
	if memoryPageDiffer(cpu6502.registers.Pc, info.OperandAddress) {
		cpu6502.opCyclesLeft++
	}

	return false
}

/*
	BCC  Branch on Carry Clear

	branch on C = 0                  N Z C I D V
									- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BCC oper      90    2     2**
*/
func (cpu6502 *Cpu6502) bcc(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.CarryFlag() == 0 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BCS  Branch on Carry Set

	branch on C = 1                 N Z C I D V
									- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BCS oper      B0    2     2**
*/
func (cpu6502 *Cpu6502) bcs(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.CarryFlag() == 1 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BEQ  Branch on Result Zero

	branch on Z = 1             N Z C I D V
								- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BEQ oper      F0    2     2**
*/
func (cpu6502 *Cpu6502) beq(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.ZeroFlag() == 1 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BIT  Test Bits in Memory with Accumulator

	bits 7 and 6 of operand are transferred to bit 7 and 6 of SR (N,V);
	the zeroflag is set to the result of operand AND Accumulator.
	The result is not kept.

	A AND M, M7 -> N, M6 -> V        N Z C I D V
									M7 + - - - M6

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	zeropage      BIT oper      24    2     3
	Absolute      BIT oper      2C    3     4
*/
func (cpu6502 *Cpu6502) bit(info cpu.OperationMethodArgument) bool {
	value := cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.UpdateNegativeFlag(value)
	cpu6502.registers.SetOverflowFlag((value>>6)&0x01 == 1)
	cpu6502.registers.UpdateZeroFlag(value & cpu6502.registers.A)

	return false
}

/*
	BMI  Branch on Result Minus

	branch on N = 1                 N Z C I D V
									- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BMI oper      30    2     2**
*/
func (cpu6502 *Cpu6502) bmi(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.NegativeFlag() == 1 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BNE  Branch on Result not Zero

	branch on Z = 0                  N Z C I D V
									- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BNE oper      D0    2     2**
*/
func (cpu6502 *Cpu6502) bne(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.ZeroFlag() == 0 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BPL  Branch on Result Plus

	branch on N = 0             N Z C I D V
								- - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BPL oper      10    2     2**
*/
func (cpu6502 *Cpu6502) bpl(info cpu.OperationMethodArgument) bool {
	//if !cpu6502.Registers.NegativeFlag {
	if cpu6502.registers.NegativeFlag() == 0 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BRK Force Break
	The BRK Instruction forces the generation of an interrupt request.
    The program counter and processor status are pushed on the stack then
    the IRQ interrupt vector at $FFFE/F is loaded into the PC and the break
    flag in the status set to one.

	interrupt,                       N Z C I D V
	push PC+2, push SR               - - - 1 - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       BRK           00    1     7
*/
func (cpu6502 *Cpu6502) brk(info cpu.OperationMethodArgument) bool {
	// Store PC in stack
	pc := cpu6502.registers.Pc + 1
	cpu6502.pushStack(types.HighNibble(pc))
	cpu6502.pushStack(types.LowNibble(pc))

	// Push status with Break flag set
	cpu6502.pushStack(cpu6502.registers.Status | 0b00010000)

	cpu6502.registers.SetInterruptFlag(true)

	cpu6502.registers.Pc = cpu6502.read16(0xFFFE)

	return false
}

/*
	BVC  Branch on Overflow Clear
	branch on V = 0               N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BVC oper      50    2     2**
*/
func (cpu6502 *Cpu6502) bvc(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.OverflowFlag() == 0 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	BVS  Branch on Overflow Set
	branch on V = 1               N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Relative      BVC oper      70    2     2**
*/
func (cpu6502 *Cpu6502) bvs(info cpu.OperationMethodArgument) bool {
	if cpu6502.registers.OverflowFlag() == 1 {
		cpu6502.addBranchCycles(info)
		cpu6502.registers.Pc = info.OperandAddress
	}

	return false
}

/*
	CLC  Clear Carry Flag
	0 -> C                        N Z C I D V
								  - - 0 - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       CLC           18    1     2
*/
func (cpu6502 *Cpu6502) clc(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetCarryFlag(false)

	return false
}

/*
	CLD  Clear Decimal Mode
	0 -> D                        N Z C I D V
								  - - - - 0 -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       CLD           D8    1     2
*/
func (cpu6502 *Cpu6502) cld(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetDecimalFlag(false)

	return false
}

/*
	CLI  Clear Interrupt Disable Bit
	0 -> I                        N Z C I D V
								  - - - 0 - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       CLI           58    1     2
*/
func (cpu6502 *Cpu6502) cli(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetInterruptFlag(false)

	return false
}

/*
	CLV  Clear Overflow Flag
	0 -> V                        N Z C I D V
								  - - - - - 0

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       CLV           B8    1     2
*/
func (cpu6502 *Cpu6502) clv(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetOverflowFlag(false)

	return false
}

/*
	CMP (CoMPare Accumulator)

	Affects Flags: S Z C

	MODE           SYNTAX       HEX LEN TIM
	Immediate     CMP #$44      $C9  2   2
	Zero Page     CMP $44       $C5  2   3
	Zero Page,X   CMP $44,X     $D5  2   4
	Absolute      CMP $4400     $CD  3   4
	Absolute,X    CMP $4400,X   $DD  3   4+
	Absolute,Y    CMP $4400,Y   $D9  3   4+
	Indirect,X    CMP ($44,X)   $C1  2   6
	Indirect,Y    CMP ($44),Y   $D1  2   5+

	+ add 1 cycle if page boundary crossed

	Compare sets flags as if a subtraction had been carried out.
    If the value in the Accumulator is equal or greater than the compared value,
    the Carry will be set. The equal (Z) and sign (S) flags will be set based on
    equality or lack thereof and the sign (i.e. A>=$80) of the Accumulator.
*/
func (cpu6502 *Cpu6502) cmp(info cpu.OperationMethodArgument) bool {
	operand := cpu6502.memory.Read(info.OperandAddress)
	cpu6502.compare(cpu6502.registers.A, operand)

	return false
}

/*
	CPX (ComPare X register)

	Affects Flags: S Z C

	MODE           SYNTAX       HEX LEN TIM
	Immediate     CPX #$44      $E0  2   2
	Zero Page     CPX $44       $E4  2   3
	Absolute      CPX $4400     $EC  3   4
*/
func (cpu6502 *Cpu6502) cpx(info cpu.OperationMethodArgument) bool {
	operand := cpu6502.memory.Read(info.OperandAddress)
	cpu6502.compare(cpu6502.registers.X, operand)

	return false
}

/*
	CPY (ComPare Y register)

	Affects Flags: S Z C

	MODE           SYNTAX       HEX LEN TIM
	Immediate     CPY #$44      $C0  2   2
	Zero Page     CPY $44       $C4  2   3
	Absolute      CPY $4400     $CC  3   4
*/
func (cpu6502 *Cpu6502) cpy(info cpu.OperationMethodArgument) bool {
	operand := cpu6502.memory.Read(info.OperandAddress)
	cpu6502.compare(cpu6502.registers.Y, operand)

	return false
}

func (cpu6502 *Cpu6502) compare(register byte, operand byte) {
	substraction := register - operand

	cpu6502.registers.SetZeroFlag(false)
	cpu6502.registers.SetCarryFlag(false)
	cpu6502.registers.SetNegativeFlag(false)

	if register >= operand {
		cpu6502.registers.SetCarryFlag(true)
	}

	if register == operand {
		cpu6502.registers.SetZeroFlag(true)
	}

	cpu6502.registers.UpdateNegativeFlag(substraction)
}

func (cpu6502 *Cpu6502) dec(info cpu.OperationMethodArgument) bool {
	address := info.OperandAddress
	operand := cpu6502.memory.Read(address)

	operand--
	cpu6502.memory.Write(address, operand)

	cpu6502.registers.UpdateZeroFlag(operand)
	//if operand == 0 {
	//	cpu6502.Registers.ZeroFlag = true
	//} else {
	//	cpu6502.Registers.ZeroFlag = false
	//}
	cpu6502.registers.UpdateNegativeFlag(operand)
	//if operand == 0xFF {
	//	cpu6502.Registers.updateFlag(negativeFlag, 1)
	//} else {
	//	cpu6502.Registers.SetNegativeFlag(false)
	//}

	return false
}

func (cpu6502 *Cpu6502) dex(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.X--
	operand := cpu6502.registers.X

	cpu6502.registers.UpdateZeroFlag(operand)
	//if operand == 0 {
	//	cpu6502.Registers.ZeroFlag = true
	//} else {
	//	cpu6502.Registers.ZeroFlag = false
	//}
	cpu6502.registers.UpdateNegativeFlag(operand)
	//if operand == 0xFF {
	//	cpu6502.Registers.updateFlag(negativeFlag, 1)
	//} else {
	//	cpu6502.Registers.updateFlag(negativeFlag, )NegativeFlag = false
	//}
	return false
}

func (cpu6502 *Cpu6502) dey(info cpu.OperationMethodArgument) bool {
	operand := cpu6502.registers.Y

	operand--
	cpu6502.registers.Y = operand

	cpu6502.registers.UpdateZeroFlag(operand)
	cpu6502.registers.UpdateNegativeFlag(operand)
	//if operand == 0 {
	//	cpu6502.Registers.ZeroFlag = true
	//} else {
	//	cpu6502.Registers.ZeroFlag = false
	//}
	//
	//if operand == 0xFF {
	//	cpu6502.Registers.NegativeFlag = true
	//} else {
	//	cpu6502.Registers.NegativeFlag = false
	//}
	return false
}

/*
	EOR (bitwise Exclusive OR)
	Affects Flags: S Z
	A EOR M -> A                     N Z C I D V
                                     + + - - - -

	MODE           SYNTAX       HEX LEN TIM
	Immediate     EOR #$44      $49  2   2
	Zero Page     EOR $44       $45  2   3
	Zero Page,X   EOR $44,X     $55  2   4
	Absolute      EOR $4400     $4D  3   4
	Absolute,X    EOR $4400,X   $5D  3   4+
	Absolute,Y    EOR $4400,Y   $59  3   4+
	Indirect,X    EOR ($44,X)   $41  2   6
	Indirect,Y    EOR ($44),Y   $51  2   5+

	+ add 1 cycle if page boundary crossed
*/
func (cpu6502 *Cpu6502) eor(info cpu.OperationMethodArgument) bool {
	value := cpu6502.memory.Read(info.OperandAddress)

	cpu6502.registers.A = cpu6502.registers.A ^ value
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)

	return true
}

/*
	INC  Increment Memory by One
	M + 1 -> M                    N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	zeropage      INC oper      E6    2     5
	zeropage,X    INC oper,X    F6    2     6
	Absolute      INC oper      EE    3     6
	Absolute,X    INC oper,X    FE    3     7
*/
func (cpu6502 *Cpu6502) inc(info cpu.OperationMethodArgument) bool {
	value := cpu6502.memory.Read(info.OperandAddress)
	value += 1

	cpu6502.memory.Write(info.OperandAddress, value)
	cpu6502.registers.UpdateZeroFlag(value)
	cpu6502.registers.UpdateNegativeFlag(value)

	return false
}

/*
	INX  Increment Index X by One
	X + 1 -> X                N Z C I D V
							  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       INX           E8    1     2
*/
func (cpu6502 *Cpu6502) inx(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.X += 1
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.X)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.X)

	return false
}

/*
	INY  Increment Index Y by One
	Y + 1 -> Y                    N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       INY           C8    1     2
*/
func (cpu6502 *Cpu6502) iny(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.Y += 1
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.Y)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.Y)

	return false
}

/*
	JMP  Jump to New Location
	(PC+1) -> PCL                    N Z C I D V
	(PC+2) -> PCH                    - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Absolute      JMP oper      4C    3     3
	Indirect      JMP (oper)    6C    3     5
*/
func (cpu6502 *Cpu6502) jmp(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.Pc = info.OperandAddress

	return false
}

/*
	JSR  Jump to New Location Saving Return types.Address
	push (PC+2),                     N Z C I D V
	(PC+1) -> PCL                    - - - - - -
	(PC+2) -> PCH

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Absolute      JSR oper      20    3     6
*/
func (cpu6502 *Cpu6502) jsr(info cpu.OperationMethodArgument) bool {
	pc := cpu6502.registers.Pc - 1
	cpu6502.pushStack(byte(pc >> 8))
	cpu6502.pushStack(byte(pc & 0xFF))

	cpu6502.registers.Pc = info.OperandAddress

	return false
}

/*
	LDA  Load Accumulator with Memory
	M -> A                        N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Immediate     LDA #oper     A9    2     2
	zeropage      LDA oper      A5    2     3
	zeropage,X    LDA oper,X    B5    2     4
	Absolute      LDA oper      AD    3     4
	Absolute,X    LDA oper,X    BD    3     4*
	Absolute,Y    LDA oper,Y    B9    3     4*
	(Indirect,X)  LDA (oper,X)  A1    2     6
	(Indirect),Y  LDA (oper),Y  B1    2     5*
*/
func (cpu6502 *Cpu6502) lda(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.A = cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)

	return true
}

/*
	LDX  Load Index X with Memory
	M -> X                    N Z C I D V
							  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Immediate     LDX #oper     A2    2     2
	zeropage      LDX oper      A6    2     3
	zeropage,Y    LDX oper,Y    B6    2     4
	Absolute      LDX oper      AE    3     4
	Absolute,Y    LDX oper,Y    BE    3     4*
*/
func (cpu6502 *Cpu6502) ldx(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.X = cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.X)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.X)

	return true
}

/*
	LDY  Load Index Y with Memory
	M -> Y                N Z C I D V
						  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	immidiate     LDY #oper     A0    2     2
	zeropage      LDY oper      A4    2     3
	zeropage,X    LDY oper,X    B4    2     4
	Absolute      LDY oper      AC    3     4
	Absolute,X    LDY oper,X    BC    3     4*
*/
func (cpu6502 *Cpu6502) ldy(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.Y = cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.Y)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.Y)

	return true
}

/*
	LSR  Shift One Bit Right (Memory or Accumulator)
	0 -> [76543210] -> C      N Z C I D V
							  0 + + - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Accumulator   LSR A         4A    1     2
	zeropage      LSR oper      46    2     5
	zeropage,X    LSR oper,X    56    2     6
	Absolute      LSR oper      4E    3     6
	Absolute,X    LSR oper,X    5E    3     7
*/
func (cpu6502 *Cpu6502) lsr(info cpu.OperationMethodArgument) bool {
	var value byte
	if info.AddressMode == cpu.Implicit {
		value = cpu6502.registers.A
	} else {
		value = cpu6502.memory.Read(info.OperandAddress)
	}

	//cpu6502.Registers.CarryFlag = value & 0x01
	cpu6502.registers.SetCarryFlag(value&0x01 == 1)

	value >>= 1
	cpu6502.registers.UpdateZeroFlag(value)
	cpu6502.registers.UpdateNegativeFlag(0)

	if info.AddressMode == cpu.Implicit {
		cpu6502.registers.A = value
	} else {
		cpu6502.memory.Write(info.OperandAddress, value)
	}

	return false
}

/*
	NOP  No Operation
	---                           N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       NOP           EA    1     2
*/
func (cpu6502 *Cpu6502) nop(info cpu.OperationMethodArgument) bool {
	return false
}

/*
	ORA  OR Memory with Accumulator
	A OR M -> A               N Z C I D V
							  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	immidiate     ORA #oper     09    2     2
	zeropage      ORA oper      05    2     3
	zeropage,X    ORA oper,X    15    2     4
	Absolute      ORA oper      0D    3     4
	Absolute,X    ORA oper,X    1D    3     4*
	Absolute,Y    ORA oper,Y    19    3     4*
	(Indirect,X)  ORA (oper,X)  01    2     6
	(Indirect),Y  ORA (oper),Y  11    2     5*
*/
func (cpu6502 *Cpu6502) ora(info cpu.OperationMethodArgument) bool {
	value := cpu6502.memory.Read(info.OperandAddress)
	cpu6502.registers.A |= value
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)

	return true
}

/*
	PHA  Push Accumulator on Stack
	push A                        N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       PHA           48    1     3
*/
func (cpu6502 *Cpu6502) pha(info cpu.OperationMethodArgument) bool {
	cpu6502.pushStack(cpu6502.registers.A)

	return false
}

/*
	PHP  Push Processor Status on Stack
	push SR                       N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       PHP           08    1     3
*/
func (cpu6502 *Cpu6502) php(info cpu.OperationMethodArgument) bool {
	value := cpu6502.registers.StatusRegister()
	value |= 0b00110000
	cpu6502.pushStack(value)

	return false
}

/*
	PLA  Pull Accumulator from Stack
	pull A                        N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       PLA           68    1     4
*/
func (cpu6502 *Cpu6502) pla(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.A = cpu6502.popStack()
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)

	return false
}

/*
	PLP  Pull Processor Status from Stack
	pull SR                       N Z C I D V
								  from stack

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       PLP           28    1     4
*/
func (cpu6502 *Cpu6502) plp(info cpu.OperationMethodArgument) bool {
	value := cpu6502.popStack()

	// From http://nesdev.com/the%20%27B%27%20flag%20&%20BRK%20instruction.txt
	// ...when the flags are restored (via PLP or RTI), the B bit is discarded.
	// From https://wiki.nesdev.com/w/index.php/Status_flags
	// ...two instructions (PLP and RTI) pull a byte from the stack and set all the flags.
	// They ignore bits 5 and 4.
	cpu6502.registers.LoadStatusRegisterIgnoring5and4(value)

	return false
}

/*
	ROL  Rotate One Bit Left (Memory or Accumulator)
	C <- [76543210] <- C          N Z C I D V
								  + + + - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Accumulator   ROL A         2A    1     2
	zeropage      ROL oper      26    2     5
	zeropage,X    ROL oper,X    36    2     6
	Absolute      ROL oper      2E    3     6
	Absolute,X    ROL oper,X    3E    3     7
*/
func (cpu6502 *Cpu6502) rol(info cpu.OperationMethodArgument) bool {
	var newCarry byte
	var value byte
	if info.AddressMode == cpu.Implicit {
		newCarry = cpu6502.registers.A & 0x80 >> 7
		cpu6502.registers.A <<= 1
		cpu6502.registers.A |= cpu6502.registers.CarryFlag()
		value = cpu6502.registers.A
	} else {
		value = cpu6502.memory.Read(info.OperandAddress)
		newCarry = value & 0x80 >> 7
		value <<= 1
		value |= cpu6502.registers.CarryFlag()
		cpu6502.memory.Write(info.OperandAddress, value)
	}

	cpu6502.registers.UpdateNegativeFlag(value)
	cpu6502.registers.UpdateZeroFlag(value)
	cpu6502.registers.SetCarryFlag(newCarry == 1)

	return false
}

/*
	ROR  Rotate One Bit Right (Memory or Accumulator)
	C -> [76543210] -> C          N Z C I D V
								  + + + - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	Accumulator   ROR A         6A    1     2
	zeropage      ROR oper      66    2     5
	zeropage,X    ROR oper,X    76    2     6
	Absolute      ROR oper      6E    3     6
	Absolute,X    ROR oper,X    7E    3     7
*/
func (cpu6502 *Cpu6502) ror(info cpu.OperationMethodArgument) bool {
	var newCarry byte
	var value byte
	if info.AddressMode == cpu.Implicit {
		newCarry = cpu6502.registers.A & 0x01
		cpu6502.registers.A >>= 1
		cpu6502.registers.A |= cpu6502.registers.CarryFlag() << 7
		value = cpu6502.registers.A
	} else {
		value = cpu6502.memory.Read(info.OperandAddress)
		newCarry = value & 0x01
		value >>= 1
		value |= cpu6502.registers.CarryFlag() << 7
		cpu6502.memory.Write(info.OperandAddress, value)
	}

	cpu6502.registers.UpdateNegativeFlag(value)
	cpu6502.registers.UpdateZeroFlag(value)
	cpu6502.registers.SetCarryFlag(newCarry == 1)

	return false
}

/*
	RTI  Return from Interrupt

	pull SR, pull PC              N Z C I D V
								  from stack

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       RTI           40    1     6
*/
func (cpu6502 *Cpu6502) rti(info cpu.OperationMethodArgument) bool {
	statusRegister := cpu6502.popStack()
	cpu6502.registers.LoadStatusRegisterIgnoring5and4(statusRegister)

	lsb := cpu6502.popStack()
	msb := cpu6502.popStack()
	cpu6502.registers.Pc = types.CreateAddress(lsb, msb)

	return false
}

/*
	RTS  Return from Subroutine
	pull PC, PC+1 -> PC           N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       RTS           60    1     6
*/
func (cpu6502 *Cpu6502) rts(info cpu.OperationMethodArgument) bool {
	lsb := cpu6502.popStack()
	msb := cpu6502.popStack()
	cpu6502.registers.Pc = types.CreateAddress(lsb, msb) + 1

	return false
}

/*
	SBC  Subtract Memory from Accumulator with Borrow
	A - M - C -> A                N Z C I D V
								  + + + - - +

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	immidiate     SBC #oper     E9    2     2
	zeropage      SBC oper      E5    2     3
	zeropage,X    SBC oper,X    F5    2     4
	Absolute      SBC oper      ED    3     4
	Absolute,X    SBC oper,X    FD    3     4*
	Absolute,Y    SBC oper,Y    F9    3     4*
	(Indirect,X)  SBC (oper,X)  E1    2     6
	(Indirect),Y  SBC (oper),Y  F1    2     5*
*/
func (cpu6502 *Cpu6502) sbc(info cpu.OperationMethodArgument) bool {
	value := cpu6502.memory.Read(info.OperandAddress)
	borrow := (1 - cpu6502.registers.CarryFlag()) & 0x01 // == !CarryFlag
	a := cpu6502.registers.A
	result := a - value - borrow
	cpu6502.registers.A = result

	cpu6502.registers.UpdateZeroFlag(byte(result))
	cpu6502.registers.UpdateNegativeFlag(byte(result))

	// Set overflow flag
	if (a^cpu6502.registers.A)&0x80 != 0 && (a^value)&0x80 != 0 {
		cpu6502.registers.SetOverflowFlag(true)
	} else {
		cpu6502.registers.SetOverflowFlag(false)
	}

	if int(a)-int(value)-int(borrow) < 0 {
		cpu6502.registers.SetCarryFlag(false)
	} else {
		cpu6502.registers.SetCarryFlag(true)
	}

	return true
}

/*
	SEC  Set Carry Flag
	1 -> C                        N Z C I D V
								  - - 1 - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       SEC           38    1     2
*/
func (cpu6502 *Cpu6502) sec(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetCarryFlag(true)

	return false
}

/*
	SED  Set Decimal Flag
	1 -> D                    N Z C I D V
							  - - - - 1 -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       SED           F8    1     2
*/
func (cpu6502 *Cpu6502) sed(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetDecimalFlag(true)

	return false
}

func (cpu6502 *Cpu6502) sei(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.SetInterruptFlag(true)

	return false
}

func (cpu6502 *Cpu6502) sta(info cpu.OperationMethodArgument) bool {
	cpu6502.memory.Write(info.OperandAddress, cpu6502.registers.A)

	return false
}

func (cpu6502 *Cpu6502) stx(info cpu.OperationMethodArgument) bool {
	cpu6502.memory.Write(info.OperandAddress, cpu6502.registers.X)

	return false
}

func (cpu6502 *Cpu6502) sty(info cpu.OperationMethodArgument) bool {
	cpu6502.memory.Write(info.OperandAddress, cpu6502.registers.Y)

	return false
}

/*
	TAX  Transfer Accumulator to Index X
	A -> X                        N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       TAX           AA    1     2
*/
func (cpu6502 *Cpu6502) tax(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.X = cpu6502.registers.A
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.X)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.X)

	return false
}

/*
	TAY  Transfer Accumulator to Index Y
	A -> Y                    N Z C I D V
							  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       TAY           A8    1     2
*/
func (cpu6502 *Cpu6502) tay(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.Y = cpu6502.registers.A
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.Y)
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.Y)

	return false
}

/*
	TSX  Transfer Stack Pointer to Index X
	SP -> X                       N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       TSX           BA    1     2
*/
func (cpu6502 *Cpu6502) tsx(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.X = cpu6502.Registers().Sp
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.X)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.X)

	return false
}

/*
	TXA  Transfer Index X to Accumulator
	X -> A                        N Z C I D V
								  + + - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       TXA           8A    1     2
*/
func (cpu6502 *Cpu6502) txa(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.A = cpu6502.registers.X
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)

	return false
}

/*
	TXS  Transfer Index X to Stack Pointer
	X -> SP                       N Z C I D V
								  - - - - - -

	addressing    assembler    opc  bytes  cycles
	--------------------------------------------
	implied       TXS           9A    1     2
*/
func (cpu6502 *Cpu6502) txs(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.Sp = cpu6502.registers.X

	return false
}

/*
	TYA  Transfer Index Y to Accumulator
	 Y -> A                           N Z C I D V
									  + + - - - -
	 addressing    assembler    opc  bytes  cycles
	 --------------------------------------------
	 implied       TYA           98    1     2

	*  add 1 to cycles if page boundery is crossed
	** add 1 to cycles if branch occurs on same page
	 add 2 to cycles if branch occurs to different page


	 Legend to Flags:  + .... modified
					   - .... not modified
					   1 .... set
					   0 .... cleared
					  M6 .... memory bit 6
					  M7 .... memory bit 7


	Note on assembler syntax:
	Most assemblers employ "OPC *oper" for forced zeropage addressing.
*/
func (cpu6502 *Cpu6502) tya(info cpu.OperationMethodArgument) bool {
	cpu6502.registers.A = cpu6502.registers.Y
	cpu6502.registers.UpdateZeroFlag(cpu6502.registers.A)
	cpu6502.registers.UpdateNegativeFlag(cpu6502.registers.A)

	return false
}
