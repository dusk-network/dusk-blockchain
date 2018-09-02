## Public Address Specification
privatekey = {}(32 byte)

publickey = ECpoint(privatekey) = {}(65 byte)

publickeyhash = ripemd160(sha3(publickey)) = {}(20 byte)

doublehash = sha3(sha3(publickeyhash)) = {}(32 byte)

checksum = doublehash[0:4] = {}(4 byte)

prefix = {}(? byte)

publicaddress = prefix + publickeyhash + checksum = {}(? byte)

publicaddressWIF = base58(publicaddress) = {}(? byte)
