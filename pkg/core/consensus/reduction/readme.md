## Binary Reduction

The Binary Reduction algorithm is an adaptation of an algorithm defined by Turpin and Coan, with two variations of the algorithm forming the core of SBA. Unlike other implementations, which normally utilize the original algorithm (for instance in extension of BBA⋆), Binary Reduction follows a two-step approach, with the input of the second step depending on the output of the first one.

If no consensus have been reached on a uniform value, the algorithm returns a default value and waits for the next instantiation.

### API

    - LaunchBlockReducer(eventbus, committee, duration) - Launches a Block Reducer broker publishing on the `OutgoingBlockAgreementTopic` topic
    - LaunchSigSetReducer(eventbus, committee, duration) - Launches a Block Reducer broker publishing on the `OutgoingSigSetAgreementTopic` topic
