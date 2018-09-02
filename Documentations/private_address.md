## Private Address Documentation
seed = {}(32 byte)

doublehash = sha3(sha3(seed)) = {}(32 byte)

checksum = doublehash[0:4] = {}(4 byte)

prefix = {}(? byte)

privatekey = prefix + seed + checksum = {}(? byte)

privateaddress = base58(privatekey) = {}(? byte)


