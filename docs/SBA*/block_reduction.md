# Block Generation Phase
### Prerequisites
A node is eligible to become a *Provisioner* once it submits a *stake* and waits for the *stake* to *mature* for `k` rounds.  
  
*Stake* structure will be outlined in a different document.
### Protocol
*Provisioner* waits until `B`<sup>`r-1`</sup> (Block `B` from round `r-1`) is *received* and *validated* (both the contents and the certificate) before starting a *timer* set for candidate propagation.

