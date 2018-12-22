package rangeproof

// // Key represents a scalar or point
// type Key [32]byte

// // add two points together
// func AddKeys(sum, k1, k2 *Key) {
// 	a := k1.ToExtended()
// 	b := new(CachedGroupElement)
// 	k2.ToExtended().ToCached(b)
// 	c := new(CompletedGroupElement)
// 	geAdd(c, a, b)
// 	tmp := new(ExtendedGroupElement)
// 	c.ToExtended(tmp)
// 	tmp.ToBytes(sum)
// 	return
// }

// // compute a*G + b*B
// func AddKeys2(result, a, b, B *Key) {
// 	BPoint := B.ToExtended()
// 	RPoint := new(ProjectiveGroupElement)
// 	GeDoubleScalarMultVartime(RPoint, b, BPoint, a)
// 	RPoint.ToBytes(result)
// 	return
// }

// // subtract two points A - B
// func SubKeys(diff, k1, k2 *Key) {
// 	a := k1.ToExtended()
// 	b := new(CachedGroupElement)
// 	k2.ToExtended().ToCached(b)
// 	c := new(CompletedGroupElement)
// 	geSub(c, a, b)
// 	tmp := new(ExtendedGroupElement)
// 	c.ToExtended(tmp)
// 	tmp.ToBytes(diff)
// 	return
// }

// func (k *Key) ToExtended() (result *ExtendedGroupElement) {
// 	result = new(ExtendedGroupElement)
// 	result.FromBytes(k)
// 	return
// }

// func identity() (result *Key) {
// 	result = new(Key)
// 	result[0] = 1
// 	return
// }

// // convert a uint64 to a scalar
// func d2h(val uint64) (result *Key) {
// 	result = new(Key)
// 	for i := 0; val > 0; i++ {
// 		result[i] = byte(val & 0xFF)
// 		val /= 256
// 	}
// 	return
// }

// // multiply a scalar by H (second curve point of Pedersen Commitment)
// func ScalarMultH(scalar *Key) (result *Key) {
// 	h := new(ExtendedGroupElement)
// 	h.FromBytes(&H)
// 	resultPoint := new(ProjectiveGroupElement)
// 	GeScalarMult(resultPoint, scalar, h)
// 	result = new(Key)
// 	resultPoint.ToBytes(result)
// 	return
// }
