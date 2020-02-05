.ORIG x3000
AND R0, R0, 0                      ; clear R0
LOOP                               ; label at the top of our loop
ADD R0, R0, 1                      ; add 1 to R0 and store back in R0
ADD R1, R0, -10                    ; subtract 10 from R0 and store back in R1
BRn LOOP                           ; go back to LOOP if the result was negative
HALT
.END
