# Provisioners and Sortition

This package implements the data structure which holds the Provisioner committee, and implements methods on top of this committee in order to be able to extract **voting committees** which are eligible to decide on blocks during the SBA\* consensus protocol.

## Abstract

Deterministic sortition is an optimization of cryptographic sortition introduced by Micali et al. It extends the functionality of cryptographic sortition in a Random Oracle setting in a non-interactive fashion, improving both the network throughput and space-efficiency.

Deterministic sortition is an algorithm that recursively hashes the public seed with situational parameters of each step, mapping the outcome to the current stakes of the Provisioners in order to extract a pseudo-random `Committee`, per step.

