# Generating A DUSK Stealth Address

    This implementation follows the DUSK whitepaper v0.3.

## Curve 

    As of writing, DUSK uses the ed25519 curve.

## Keys 

    Private keys in DUSK are generated using 32 bytes of entropy. There are two keypairs called spend and view as opposed to one. 

## Generating Stealth Address from seed

    We take the seed which is 256 bits or 32 bytes.

    We then check whether the seed `s` is a valid ed25519 scalar, by ensuring s E [0, l) where l is the curve order. If it is not, we reduce the scalar using ScReduce32 and the result is our `Private Spend Key`. This is checked with the IsLessThanOrder function.

    The Private View Key is derived by hashing the `Private Spend Key` using Keccak256. The result is then checked if it also needs to be reduced, and stored as the `Private View Key`.

    Note: The checking process is normally omitted, and future implementations may omit this. The reason is because key generation is not normally a place where speed is generally needed unless you are batching, plus with it removed the readablity would be improved. Noting that if k < .

    The `Public View Key` and `Public Spend Key` are generated the same way; by multiplying their private counterparts with the basepoint. The function PrivateToPublic() demonstrates this.

    The stealth Address:

        Let netPrefix = 0xEF

        Let t = netPrefix + PublicSpendKey + PublicViewKey 

        Let checksum = Keccak256(t)[:4]

        StealthAddress = Base58(t + checksum)


### EC - normal public/private keys -- TBD

Below I will write an overview of the elliptic curve.

We choose a `b`, where b >= 10. 

    In our case b = 256.

We choose a cryptographic hashing function, such that output is 2b in length.
If b = 256, we must choose a hashing algorithm which produces 512 bits. 

    In our case, we choose SHA512.

We choose a number `q` which is a prime power that is also congurent to 1 mod 4. A prime power is a number which is divisible by only one prime number. For example 9 is a prime power because it is only divisible by the prime number 3.

    In our case q = 2**255 - 19.

Checking wolfphram alpha you can check it's primality and by consequence determine that is a prime power: http://www.wolframalpha.com/widgets/view.jsp?id=ccbaefcc48cd5f8ec9309165ea694eb2

To check if this number is also congruent to 1 mod 4, we will use the rule that:

    ` a + b mod n = (a modn + b modn) mod n`

    Let a = 2**255 , b = -19, n = 4

    a modn = 2**255 mod4 = 0

    b modn = -19 mod4 = 1

    Therefore: 
    
    2*255 - 19 mod4 = (1+0) mod4 = 1 mod4 

We choose a prime l such that 2**(b-4) < l < 2**(b-3)

    In our case l = 2**252 + 27742317777372353535851937790883648493

We choose a non-square element d in the finite field q. 

