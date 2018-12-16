## Notes


Currently the blinding factor is generated in the prove function
Once everything work, we can change the API to have the blinding factor as an argument
This would involve changing the pedersen package as well as the bulletproof package

// We need a better API for fetching scalar values such as one and zero

// Note that v is actually a ristretto.Scalar, which means it is a v mod p and not v itself. I'm sure this would not be a problem as p is large, but would like to sanity check.

// When finished, double check that all types are correct e.g r0 should be []ristretto.Scalar instead of []ristretto.Point

// When finished, need checks for if any of the random challenges(x,y,z...) are zero

// Need to check that we are changing the bases where necessary for the pedersen commitments (FIXED)

// Helper function for committing to multiple vectors (FIXED)

// fix the api for errors; some have errors and others do not, check ones that do and remove any that could never have them. Length errors can be fixed by using arrays instead of slices.

// Fix abstraction for pedersen; deterministically calculate the commitment to different vectors, based on the first generator label - we could just keep hashing, or we could append a different number (FIXED)

// Double check we calculated the fiat shamir properly; N.B. Java implementation appends and then converts and the intermediate using HashToScalar immediately. While this implementation, appends and only uses hash to scalar when we need to construct a challenge (FIXED)

// Possible BUG: when pedersen commits to a vector, it changes the generator from the initial to a new one (FIXED)

// Can we put the t0 checks in a better place? We don't need all of them, and I think all of them should only be there for debugging.

// For now the pedersen structure, is recreated within each function for simplicity. Once all tests pass, we will have it as a global parameter within the rangeproof package. If localised to only the prove function, then we will need to re-calculate all bases every time the prove or verify method is instantiated

/* Possible BUG in ristretto lib

P = some number point
X = ristretto.Point{}

Adding X to P, will result in zeros

Fixed by setting X = X.SetZero()


Also

P = some point that is not zero

P.Equals(ristretto.Point{}) seems to always return true

when you just declare a point in ristretto, then attempt to use a method like .Add()
It will not add. You need to use setZero() on it. AFAIK, this is not a problem with the scalar struct

*/

TODO:

- Fix ensures and debug statements
- Fix Pedersen API with VecExp; also add blinding factor

For the ugly sum/concat in r(x) l71, we could cache the values from m = 1 to m = 10 in memory

For the bitcommitment, i it faster to just calculate aL, then do a vectorSub to get aR?


Using slices now, cause a lot of allocations to be done...

Pass around precomputed slices such as z^M, and 2^N

When done find all "_" for ommitted errors


BUG maybe: Pedersen.go l129 clash with generator in pedersen and iterate in generator 

Performance: make([]int, 0) and []int{} are the same and require the compiler to constantly allocate memory


Double check where each ristreto.Point has been declared, due to what happens when we do not set it zero before using it.
    Remember to notify or send a bug fix to original repo

Change the way the generate generates points, by using the previous generators, bytes in the Derive method (DONE)

Bug when y is set to zero for RxH debug  (Although the proof should not pass if the challenge is zero)


// Deal with Padding when m is not multiple of two
// Convert moduled check into one check for verify (0 = check1 * c + check2)
// Performance, pass the pedersen struct to the Prove and Verify as argument, then pass it down to each function that needs it. We can alternatively pass the slice of vector points instead to the functions inside of prove and verify that need the slices.


N.B. libsecp256 lib uses a different nonce for blinding point and one for other points. Doing this could potentilly
make the API better in terms of G[1:]