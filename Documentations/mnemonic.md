### Mnemonic Phrase Specification
Based on [BIP-39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki) (Bitcoin Improvement Proposal) specification.

The specification below highlights the use of 32 byte seed, however, an extended private key can be generated with various seed sizes (16 byte, 20 byte, 24 byte, 28 byte and 32 byte) with a varying checksum size as a result (4 bit, 5 bit, 6 bit, 7 bit and 8 bit).

**Mnemonic Phrase Generation:**

seed = {}(32 byte)

doublehash = sha3(sha3(seed)) = {}(32 byte)

checksum = doublehash[0:1] = {}(1 byte)

mnemonicsequence = seed + checksum = {}(33 byte)

mnemonicphrase = (mnemonicsequence / 24) = 24 * 11 bit segments = {}(33 byte)

**Extended Private Key Derivation:**

salt = "mnemonic" + password = {}(? byte)

extendedprivatekey = pbkdf2_hmac(mnemonicsequence + salt; function = sha512; roundnumber = 2048) = {}(64 byte)



