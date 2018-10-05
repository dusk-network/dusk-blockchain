# Generating A DUSK Stealth Address

    This implementation follows the DUSK whitepaper v0.3.

## Curve 

    As of writing, DUSK uses the ed25519 curve.

## Keys 

    Private keys in DUSK are generated using 32 bytes of entropy. There are two keypairs called spend and view as opposed to one. 

## Generating Stealth Address from seed

    We take the seed which is 256 bits or 32 bytes.

    We then check whether the seed `s` is a valid ed25519 scalar, by ensuring s E [0, l) where l is the curve order. If it is not, we reduce the scalar using ScReduce32 and the result is our `Private Spend Key`. The validity is checked with the IsLessThanOrder function.

                    s = randomEntropy(32)
                    if !isLessThanCurveOrder(s) {
                        s = reduce(s)
                    }
                    PrivateSpendKey = s

    The Private View Key is derived by hashing the `Private Spend Key` using Keccak256. The result is then checked if it also needs to be reduced, and stored as the `Private View Key`.

                    PrivateViewKey = Keccak256(PrivateSpend)
                    if !isLessThanCurveOrder(PrivateViewKey) {
                        PrivateViewKey = reduce(PrivateViewKey)
                    }

    Note: The checking process is normally omitted, and future implementations may omit this. The reason is because key generation is not normally a place where speed is generally needed unless you are batching, plus with it removed the readablity would be improved.

    The `Public View Key` and `Public Spend Key` are generated the same way. This is done by multiplying their private counterparts with the basepoint G. The function PrivateToPublic() demonstrates this.

                    PublicViewKey = PrivateToPublic(PrivateViewKey)
                    PublicSpendKey = PrivateToPublic(PrivateSpendKey)

    Stealth Address:

                    netPrefix = 0xEF

                    t = netPrefix + PublicSpendKey + PublicViewKey 

                    checksum = Keccak256(t)[:4]

                    StealthAddress = Base58(t + checksum)

Note: scReduce32(x) is simply x mod l . To do this simply we would need to use divide and branches, however this implementation would leak information about the seed because division asm and branches take varying amounts of time to complete. We need a constant time.