package utils

// Search replicate sort.Search() upon uint64 type with error return
func Search(n uint64, f func(uint64) (bool, error)) (uint64, error) {
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	var i uint64
	j := n
	for i < j {
		h := uint64(i+j) >> 1 // avoid overflow when computing h
		// i ≤ h < j
		res, err := f(h)
		if err != nil {
			return 0, err
		}

		if !res {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i, nil
}
