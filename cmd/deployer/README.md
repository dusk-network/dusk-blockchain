# Deployer

Deployer is an application that simplifies the procedure of (re)starting a dusk-blockchain node (both dusk-blockchain and rusk services). It should also facilitate automatic diagnostic of runtime issues.

<!-- ToC start -->
## Contents

   1. [Rules](#rules)
   1. [How to run](#how-to-run)
<!-- ToC end -->

## Rules

| Network | Condition | Actions |
| :--- | :--- | :--- |
| DevNet | Unresponsive service | Send SIGABRT signal \| Collect core dump\| Restart with CPU profile 
| DevNet | Memory allocated > `SoftLimit_1` | Enable `memstats` profile\| Enable Trace log level
| DevNet | Memory allocated > `SoftLimit_2` | Enable `memprofile` profile|
| DevNet | Memory allocated > `HardLimit` | Send SIGABRT signal \| Collect core dump \| Restart process
 
 
## How to run

```bash
DUSK_BLOCKCHAIN_PATH=
RUSK_PATH=

deployer --config /home/.dusk/dusk.toml
```
