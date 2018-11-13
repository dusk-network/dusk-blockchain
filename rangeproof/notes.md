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