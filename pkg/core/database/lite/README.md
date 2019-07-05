### General concept
Lite package represents a database driver that provides in-memory blockchain storage. That said, it does not provide persistence storage. For now, it is applicable only for testing purposes like:

- running a unit-test
- running a benchmark test
- running a test harness

It still implements transaction layer requirements like atomicity, consistency and isolation but no durability. At some point later it can be used as baseline to compare performance results.


