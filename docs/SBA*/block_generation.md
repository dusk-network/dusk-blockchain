# Block Generation Phase
### Prerequisites
A general network node (*full-node*) is eligible to become a Block Generator (*BG*) once it submits a *blind-bid* and waits for the bid to *mature* for `k` rounds.  
  
*Blind-bid* structure will be outlined in a different document.
### Protocol
*BG* waits until `B`<sup>`r-1`</sup> (Block `B` from round `r-1`) is *received* and *validated* (both the contents and the certificate).

1. *BG* checks its inclusion in the Block Generation Phase of round `r` by calling function `F(b,Y)` where `F` is a *gamma distribution* function, `b` is the amount of DUSK
tokens contained in the *blind-bid* and `Y = H(b,H(seed`<sup>`r-1`</sup>`|H(b|H(secret))))`.
2. If the output (*score*) of `F` is smaller than the predetermined *threshold* `\tau`, *BG* forges a *candidate block* (`B`<sup>`r`</sup><sub>`candidate`</sub> = `{r,B`<sup>`r-1`</sup>`,SIG`<sub>`BLS`</sub>`(seed`<sup>`r-1`</sup>`),PAY`<sup>`r`</sup>`)`)
3. *BG* propagates the *score* alongside the *zero-knowledge proof* of the *blind-bid*.
4. *BG* propagates the *candidate block*.