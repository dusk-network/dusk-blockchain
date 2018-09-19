### Stealth Transaction

transaction = version + type + |typeattributes| (? byte)

version = 0x00 (1 byte)

type = 0x00 (1 byte)

|typeattributes| = inputcount (1 byte) + {input} (? byte) + txpublickey + outputcount (1 byte) + {output} (? byte)

input = txid + index (1 byte) + signature (? byte)

txid = {} (32 byte)

txpublickey = rG mod l (32 byte)

output = amount (8 byte) + destinationkey

destinationkey = H(rA)G + B

r = {0; 2^256}

G = Generator

l = Base Point
