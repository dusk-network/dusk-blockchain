# Ring Signature

## Introduction 

    A ring signatura es a type of signature scheme, where the signer chooses a set of people who will participate in the signature scheme with them. The set is known as the `ring`. The people chosen do not need to know that they are chosen, and do not need to actively participate in the scheme (non-interactive); the signer only needs their public keys. 

    The reason for using a ring signature which is based upon a group signature, is due to anonymity. An outside party cannot be certain which member of the ring signed the message, however they can validate that the message was signed by someone in the ring.

    This is useful in cryptocurrencies as a user can sign an input with a ring signature and without knowing who signed it, the full nodes can validate that the tx was correctly signed by a member.

    In privacy cryptocurrencies this poses a problem, because the inputs/outputs are also anonymised. If the input/output was not anonymised, then a third party could look at what input was signed and look at the corresponding address assosciated with it.

    However, if the inputs are anonymised, then the validator will not be able to check on the blockchain whether it was spent already. This can be remedied by other cryptographic advancements, however since the topic concerns basic ring signatures, we will only focus on this for now.

    Concluding this section, the purpose of the ring signature is to anonymise the sender.

## Input

    The input to a ring signature rs, is:

        - The message, `m`

        - The set of public keys of the other `n` members `pk_n` `n !=1 `

        - The private key of the signer, `sk_1`

## Output

    The output is a ring signature (rs) defined on m. `rs(m)`

    rs consists of :
    
            - The set of public keys for the ring including the signers.`pk_n`

            - The KeyImage which is a special object that marks the signer of the ring. It does not tell who it is, but if the signer tries to sign with the same key again, they will generate the same Key Image. `KI`

            - n s values

            - n c values

            s and c values are used to ensure that m was correctly signed, the only way to forge this signature would be to break the Discrete Logarithm problem or the hash function used.

## Note

    The ring signature used is MLSAG, and uses one c value to save space. The other c values are generated upon verification. Although it makes verification a little longer, it saves storage space, which I believe is a necessary evil, as you can always optimise verification times in the future.

    For more detail, please check out the implementation in the `MLSAG` package.