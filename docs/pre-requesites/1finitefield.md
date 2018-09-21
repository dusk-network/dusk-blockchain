# Finite Fields

-- Will edit more when time permits. Finite Field and ecc file will give a quick overview of elliptic curves.

Assume:

    - Basic knowledge of set theory

A simplistic definition of finite fields:

    (1) A finite field is a set of numbers, where addition and subtraction and consequently multiplication and division of an element in the set with another element in the set gives another element in the set. (closure)

    (2) The cardinality of the set is a prime power. This will ensure the operations described above are all defined correctly later.*

Take the set {0, 1, 2, 3, 4} . 

We will determine from our crude definition, if this is a Finite Field.

(1) The cardinality of the set is 5, which is a prime power. 

(2) When we add (1,0) we get one which is an element of the field. However, when we add (4,1) we get five, which is not an element of the field. Under the regular definition for addition, this is not a finite field.

Changing the definitions for our operations by adding modulo n where n is the cardinality of the field, will ensure that we have closure.

For example: (4,1) = (4 + 1) mod 5 = 0 . Using the definitions for modulo, this holds for division, multiplication and subtraction.

* For a finite field x, if x is not a prime power, then n time x where 1 <= n <= x-1 will not always be equivalent and so we would not have an inverse for some elements under multiplication. We will need the inverse under multiplication because this is how division is defined.


## Finite field multiplication and division

Over the Finite Field of order k, a times b is defined as:


    a x b mod k

For example:

    Over the finite field of order 23:

        12 * 5 = 14 (60 mod23 )

Division is the inverse of multiplication:

    If 12 * 5 = 14 then 14/5 = 12

In order to do the division calculation, we must use a formula which will not be proved here:

        Let p = the prime order of the field

        1/n = n^(p-2)

Using this formula, we can convert the inverse into a base and power:

    Over the finite field 23:

    14 / 5 = 14 * 1/5 = 14 * 5^(23-2) = 14 * 5^21 = 12
