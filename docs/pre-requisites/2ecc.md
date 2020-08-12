# 2ecc

## ECC

## Curves over Finite Fields

Finite Fields are important to ECC because normally elliptic curves are defined over prime finite fields and not over the Complex numbers. This means that given a curve not every number will be on the curve when defined over the Finite field.

For example:

If we use the quadratic curve: y = \(x^2 - 1\) for simplicity and define it over the Finite field of order 23. Then we can check if \(3,9\) is a point on this curve. Note : Over the Real numbers this would be true.

RHS: 3^2 - 1 mod23 = \(9 mod23 - 1 mod23\) mod23 = \(9 + 22\) mod23 = 8

LHS: 9 mod23 = 9

Since LHS does not equal RHS, \(3,9\) is not a point on this curve.

## Point Addition for Elliptic Curves

Given two points P and Q.

P + Q on a elliptic curve defined over a finite field where P and Q do not lie on a line with undefined gradient \(Vertical line \| \), is defined as follows

Let P = \(Px, Py\); Q = \(Qx, Qy\); R = P + Q = \(Rx, Ry\)

Gradient = \(Qy - Py\) / \(Qx - Px\) = m

Rx = m^2 - Px - Qx

Ry = m\(Px - Qx\) - Py

## Group

A group is another type of closed set under one operation, such that the set has an identity and is assosciative, commutative and invertible.

The operation in question is point addition however, we will not prove that the following properties hold.

Note: Point addition so far had nothing to do with modulo. Below we will be using point addition with Finite Fields, which will lay the foundations for our Elliptic Curve Cryptography.

Given a point G that is on the ellitpic curve, we can use point addition to add G to itself, in turn `Generating` other points until kG=0 where k is the number of times we have added G to itself.. the set of points generated with G, form a Group.

Let G be some point \(a,b\)

The group would therefore look like : {0, G, 2G, 3G, 4G...\(k-1\)G}. We go upto \(k-1\)G because kG = 0 which is the first element.

## Private Keys

Let X = kG Let prime field = p

Since k is the amount of times G has been added to itself:

```text
G^k = X mod p
```

To calculate k:

```text
Assume the log has a base of G

k = logX mod p
```

This is what is known as the discrete log problem, it has not been proved to be unsolvable, however no-one has been able to crack it yet with a large enough p. Note that if we did not have the mod p making it a continous logarithm problem, then we would be able to input the numbers in our calculator.

For this reason, `k` the `scalar` is what we refer to as the `private key` and`kG` which will yield a point since G is a point, is what we refer to as the `public key`. Therefore, given `k` the private key, we can find the public key by doing `scalar multiplication`.

Recapping:

Looking at the definition of Elliptic curves, the following are defined:

* G: which is the generator.
* n : the number of points in our group formed by the Generator
* p: The prime field. Fp
* y\(x\) : The elliptic curve equation: y = x^3 + ax + b
* a: the co-efficient for elliptic curve
* b: the co-efficient for elliptic curve
* h: Is a number which represents l/n where n is defined above and l is the order of the entire field.

