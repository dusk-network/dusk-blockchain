### Stealth Transaction

transaction = version (1 byte) + type (1 byte) + |typeattributes| (? byte)

version = 0x00

type = 0x00

|typeattributes| = inputcount (1 byte) + {input} (? byte) + outputcount (1 byte) + {output} (? byte)
