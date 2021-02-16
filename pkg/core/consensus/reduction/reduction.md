# Binary Reduction Phase

## Abstract

The Binary Reduction algorithm lays at the core of SBA\*. It converts the problem of reaching consensus on arbitrary values to reaching consensus on one of two values. It is an adaptation of the Turpin and Coan algorithm, originally concocted to solve the general Byzantine agreement when given a binary Byzantine agreement algorithm as a subroutine, for `n > 3f` \(with `n` defined as total number of nodes and `f` defined as adversarial nodes\).

Unlike other implementations, which normally utilize the original algorithm, Binary Reduction adopted in SBA\* follows a two-step approach, with the input of the second step depending on the output of the first one.

If no consensus have been reached on a uniform value, the algorithm returns a default value and waits for the next instantiation.

Binary Reduction acts as a uniform value extraction function which is then fed through the Block Agreement algorithm before exiting the loop in case of a successful termination of the Block Agreement algorithm.

