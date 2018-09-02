### Encrypted Private Key Specification
Based on [NEP-2](https://github.com/neo-project/proposals/blob/master/nep-2.mediawiki) (NEO Enhancement Proposal) specification.

**Encryption:**

privatekey = {}(32byte)

doublehash = sha3(sha3(privatekey)) = {}(32 byte)

checksum = doublehash[0:4] = {}(4 byte)

salt = checksum + password = {}(? byte)

key = scrypt(salt) = {}(64 byte)

derivedhalf1 = key[0:32] = {}(32 byte)

derivedhalf2 = key[32:64] = {}(32 byte)

encryptedhalf1 = aes256encrypt(block = privatekey[0:16] XOR derivedhalf1[0:16]; key = derivedhalf2[0:16]) = {}(16 byte)

encryptedhalf2 = aes256encrypt(block = privatekey[16:32] XOR derivedhalf1[16:32]; key = derivedhalf2[16:32]) = {}(16 byte)

prefix = {}(? byte)

encryptedprivatekey = prefix + encryptedhalf1 + encryptedhalf2 + checksum = {}(? byte)

encryptedprivatekeyWIF = base58(encryptedprivatekey) = {}(? byte)

**Decryption:**

salt = checksum + password = {}(? byte)

key = scrypt(salt) = {}(64 byte)

derivedhalf1 = key[0:32] = {}(32 byte)

derivedhalf2 = key[32:64] = {}(32 byte)

privatekeyhalf1 = aes256decrypt(block = encryptedhalf1]; key = derivedhalf2[0:16]) XOR derivedhalf1[0:16] = {}(16 byte)

privatekeyhalf2 = aes256decrypt(block = encryptedhalf2]; key = derivedhalf2[16:32]) XOR derivedhalf1[16:32] = {}(16 byte)

privatekey = privatekeyhalf1 + privatekeyhalf2 = {}(32 byte)

