# dusk-go



F :

    - Restricted devnet without block processing + more


K: 

    - Ristretto:

        - Implement the hash to point seen in CurveDalek as current impl is not uniform, we saw this in the monero repo also, where 

        - Remove vartime scalar multiplication(Do this last) 

        - Implement against the Previous `Key` abstract, we can remove co-factor * order multiplication now and checking whether in subgroup

        - Extra : Check for optimisations against the Rust curve2519 implementation. (Ongoing)

    - RingCT

        - Pederson Commitment (Hidden input/output)

        - BulletProof reference from Java with multiple inputs (Proof input/out between range) Replacing range proof

        - Ring Signature (Signing and Verification)

    - Re-Implement Stealth on top of Ristretto as there is a new API



    Note:

        Public address is not the same as the stealth address.


        This seems pretty expensive;https://github.com/monero-project/monero/commit/c42917624849daeac0b4bc2fb1cd1f2539470b28