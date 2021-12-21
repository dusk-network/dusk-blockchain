# Deployer

Deployer is an application that simplifies the procedure of (re)starting a dusk-blockchain node (both dusk-blockchain and rusk services). It should also facilitate automatic diagnostic of runtime issues.

<!-- ToC start -->
## Contents

   1. [Description](#description)
   1. [Rules](#rules)
   1. [How to run](#how-to-run)
<!-- ToC end -->

## Description

Deployer is the name here as the totality of the full node on the Dusk network
is more than one component. Deployer is the launcher that coordinates the
startup and shutdown of Dusk and Rusk, the node and the VM. It monitors to see
they are still live, restarts them if needed, and shuts them down as required.



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
