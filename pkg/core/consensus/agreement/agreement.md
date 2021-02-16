# Agreement Phase

## Abstract

During the conduction of a technical analysis of sortition based consensus algorithms, the team has discovered a vulnerability, which increases the probability of a the consensus forking, dubbed a _timeout fork_. As a result, the team has concluded that SBA requires an additional step in the inner loop to guarantee statistical finality under the basic assumptions of the protocol.

`Block Agreement` is an asynchronous algorithm running in parallel with the inner loop. Successful termination of the algorithm indicates that the relevant inner loop has been successfully executed and the protocol can proceed to the next loop. The algorithm provides a statistical guarantee that at least one honest node has received a set of votes exceeding the minimum threshold required to successfully terminate the respective phase of the protocol.
