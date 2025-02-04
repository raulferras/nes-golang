; Prefix included at the beginning of each test. Defines vectors
; and other things for custom devcart loader.
;
; Reset vector points to "reset".
; NMI points to "nmi" if defined, otherwise default_nmi.
; IRQ points to "irq" if defined, otherwise default_irq.

default_nmi:
	rti

default_irq:
	bit $4015
	rti

; Delays for almost A milliseconds (A * 0.999009524 msec)
; Preserved: X, Y
delay_msec:
	  pha                     ; 3			Push acumulator on stack
      lda   #253              ; 2			Save FD into A
      sec                     ; 2			Set carry flag

; Delays for almost 'A / 10' milliseconds (A * 0.099453968 msec)
; Preserved: X, Y
delay_msec_:
      nop                     ; 2			
      adc   #-2               ; 2			Add with carry				|
      bne   delay_msec_       ; 3			Branch on Result not Zero 	|-> loop while A-1 != 0
                              ; -1
      pla                     ; 4			Pull acumulator from stack (N Z)
      clc                     ; 2			Clear carry flag
      adc   #-1               ; 2			Add with carry -1           |
      bne   delay_msec        ; 3			Branch on Result not Zero   |-> loop while A-1 != 0
      rts
      .code

; Variable delay. All calls include comment stating number of clocks
; used, including the setup code in the caller.
delay_yaNN:

; Report value in low-mem variable 'result' as number of beeps and
; code printed to console, then jump to forever.
report_final_result:

; Disable IRQ and NMI then loop endlessly.
forever:


; Report error if last result was non-zero
error_if_ne:
	bne error_if_
	rts

; Report error if last result was zero
error_if_eq:
	beq error_if_
	rts

; Report error
error_if_:
	jmp report_final_result

