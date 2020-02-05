;; Reverse a string
        .ORIG    x3000
rev     LEA      R0,FILE      ;; R0 is beginning of string
        ADD      R1,R0,#-1
LOOP1   LDR      R3,R1,#1     ;; Note -- LDR "looks" at the word past R1
        BRz      DONE1
        ADD      R1,R1,#1
        BR       LOOP1

DONE1   NOT      R2,R0
        ADD      R2,R2,R1

;; R0 == address of first character of string
;; R1 == address of last character of string
;; R2 == size of string - 2  (Think about it....)
LOOP2   ADD      R2,R2,#0
        BRn      DONE2
        LDR      R3,R0,#0     ;; Swap
        LDR      R4,R1,#0
        STR      R4,R0,#0
        STR      R3,R1,#0
        ADD      R0,R0,#1     ;; move pointers
        ADD      R1,R1,#-1
        ADD      R2,R2,#-2    ;; decrease R2 by 2
        BR       LOOP2

DONE2   LEA R0, FILE
	PUTS
	HALT

FILE    .STRINGZ "ABCD1234"
        .END

