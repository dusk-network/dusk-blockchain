go-ristretto
============

Many cryptographic schemes need a group of prime order.  Popular and
efficient elliptic curves like (Edwards25519 of `ed25519` fame) are
rarely of prime order.  There is, however, a convenient method
to construct a prime order group from such curves,
called [Ristretto](https://ristretto.group) proposed by Mike Hamburg.

This is a pure Go implementation of the group operations on the
Ristretto prime-order group built from Edwards25519.
Documentation is on [godoc](https://godoc.org/github.com/bwesterb/go-ristretto).

Example: El-Gamal encryption
----------------------------

```go
// Generate an El-Gamal keypair
var secretKey ristretto.Scalar
var publicKey ristretto.Point

secretKey.Rand() // generate a new secret key
publicKey.ScalarMultBase(&secretKey) // compute public key

// El-Gamal encrypt a random curve point p into a ciphertext-pair (c1,c2)
var p ristretto.Point
var r ristretto.Scalar
var c1 ristretto.Point
var c2 ristretto.Point
p.Rand()
r.Rand()
c2.ScalarMultBase(&r)
c1.ScalarMult(&publicKey, &r)
c1.Add(&c1, &p)

// Decrypt (c1,c2) back to p
var blinding, p2 ristretto.Point
blinding.ScalarMult(&c2, &secretKey)
p2.Sub(&c1, &blinding)

fmt.Printf("%v", bytes.Equal(p.Bytes(), p2.Bytes()))
// Output:
// true
```


References
----------

The curve and Ristretto implementation is based on the unpublished
[PandA](https://link.springer.com/chapter/10.1007/978-3-319-04873-4_14)
library by
[Chuengsatiansup](https://perso.ens-lyon.fr/chitchanok.chuengsatiansup/),
[Ribarski](http://panceribarski.com) and
[Schwabe](https://cryptojedi.org/peter/index.shtml),
see [cref/cref.c](cref/cref.c).  The generic field operations borrow
from [Adam Langley](https://www.imperialviolet.org)'s
[ed25519](http://github.com/agl/ed25519).
The amd64 optimized field arithmetic are from George Tankersley's
[ed25519 patch](https://go-review.googlesource.com/c/crypto/+/71950),
which in turn is based on SUPERCOP's
[amd64-51-30k](https://github.com/floodyberry/supercop/tree/master/crypto_sign/ed25519/amd64-51-30k)
by Bernstein, Duif, Lange, Schwabe and Yang.
The variable-time scalar multiplication code is based on that
of [curve25519-dalek](https://github.com/dalek-cryptography/curve25519-dalek).

### other platforms
* [Rust](https://github.com/dalek-cryptography/curve25519-dalek)
