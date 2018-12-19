# Block Generation Phase
### Prerequisites
A node is eligible to become a *Provisioner* once it submits a *stake* and waits for the *stake* to *mature* for `k` rounds.  
  
*Stake* structure will be outlined in a different document.
### Protocol
*Provisioner* waits until `B`<sup>`r-1`</sup> (Block `B` from round `r-1`) is *received* and *validated* (both the contents and the certificate) before starting a *timer* set for candidate block propagation.

1. If the *Provisioner* receives a valid *candidate block* (both the contents and the *score* of the *BG*) for round `r` before the timeout of the *timer* set for candidate
block propagation, *Provisioner* starts a *timer* set for vote propagation, skips to step 2 with the *candidate block*, otherwise skipping to step 2 with an *empty block*.
2. *Provisioner* checks its inclusion in Block Reduction Phase One of round `r` by calling `F(b,Z)` where F is a gamma distribution function, b is the amount of DUSK
tokens contained in the stake and `Z = H(b|SIG`<sub>`BLS`</sub>`(seed`<sup>`r-1`</sup>`|round|step))`.
3. If the output (*score*) of `F` is smaller than the predetermined *threshold* `tau`, *Provisioner* propagates the *score* alongside a *vote* for a block selected in step 1.
4. If the *Provisioner* receives *votes* exceeding *threshold* `t` for the same block before the timeout of the *timer* set for vote propagation, *Provisioner*
starts a *timer* set for vote propagation, skips to step 5 with the block with above-*threshold* votes, otherwise skipping to step 5 with an *empty block*.
5. *Provisioner* checks its inclusion in Block Reduction Phase Two of round `r` by calling `F(b,Z)` where F is a gamma distribution function, b is the amount of DUSK
tokens contained in the stake and `Z = H(b|SIG`<sub>`BLS`</sub>`(seed`<sup>`r-1`</sup>`|round|step))`.
6. If the *Provisioner* receives *votes* exceeding *threshold* `t` for the same block before the timeout of the *timer* set for vote propagation, *Provisioner*
returns the block with above-*threshold* votes, otherwise returning an *empty block*.