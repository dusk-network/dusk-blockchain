## Week 2 - (Starting 1st October)

This week the goal was to implement Ring signatures and start bulletproofs. The ring signatures have been implemented however, it is for a single key and not the generalised MLSAG variation. The mathematics for both are similar and so this should not be a problem. The Ristretto implementation is starting to mould very nicely with the rest of the code. I do think that we should add another layer on top of the Ristretto API, so that we can swap it out for another if need be.

For example, If we add an interface for point like so:

    type Point interface {
        Add(p1, p2 *Point) *Point
        Sub(p1, p2 *Point) *Point
        Neg(p1 *Point) *Point
        Set(b []byte) *Point
    }

Then another for Scalar,Curve and Hide the ristreto implemenation behind these.

This can be done after the first release. I am doubtful that we will switch out the curve, however if we want to stick with the modular convention, it would be wise.

This week we will flesh out the general architecture for the Wallet (This may include adding the StealthTX) and go over what we have implemented so far in bulletproofs.

## Note

    After the first release, we will start doing weekly refactors, to keep code fresh, not build up technical debt and maintain readability.