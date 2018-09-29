## Week 1


The goal of this week was to go over the Ristretto implementation, check it against the paper and familiarise myself with any optimisations that may be needed. For example, the previous Ed25519 library had CacheGroupElement, which we will need to integrate in the future, in order to speed up the calculations. I would like to do this without breaking the simplicity of the Ristretto API

Will need to re-read the paper on bulletproofs again and read the reference implementation another time to see where things were done because of the Ed25119 curve. 

Pederson commitments, or Vector commitments can be done in one function. In the bulletproof repo it is:
```
func vecExp(a Key, b Key) (Key, error) {
	result := Key{} // defaults to zero

	if len(a) != len(b) {
		return Key{}, errors.New("length of scalar a does not equal length of scalar b")
	}

	if len(a) <= N*M {
		return Key{}, errors.New("length of scalar a is not less than N*M")
	}

	for i := 0; i < N; i++ {

		//res = res + Gi*a
		// res = res + Hi *b
	}

	return result, nil
}
```

This could not be implemented straight away because the Ristretto does not currently have the API for pre-computed values Hi and Gi. And instead of continuing with the old Ed25519 API, I think it best to rip it all out now instead of later.

I am not too worried about implementing Ring Signature. I have seen some implementations of it and understand the mathematics to a good degree. The bulk of my resources will and has been spent on Ristretto and Bulletproofs.

Once these components are integrated, we can then move onto the stealthtx, a basic structure has ben laid out for it, however I believe that we need to flesh out the other parts before starting on it.