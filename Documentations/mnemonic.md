### Mnemonic Phrase Specification
Based on [BIP-39](https://github.com/bitcoin/bips/blob/master/bip-0039.mediawiki) (Bitcoin Improvement Proposal) specification.

seed = {}(32 byte)

doublehash = sha3(sha3(seed)) = {}(32 byte)

checksum = doublehash[0:1] = {}(1 byte)

mnemonicsequence = seed + checksum = {}(33 byte)

mnemonicphrase = (mnemonicsequence / 24) = 24 * 11 bit segments = {}(33 byte)

salt = "mnemonic" + password = {}(? byte)

extendedprivatekey = pbkdf2_hmac(mnemonicsequence + salt; sha512; 2048 rounds) = {}(64 byte)



