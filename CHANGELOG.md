# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Add `raw` field to GQL transaction object [#1481]
- Add `blocksrange` filter to GQL transaction lookup [#1468]

### Removed
- Remove hardcoded genesis checks [#1464]
- Remove provisioners list updates at every block [#1479]

## [0.6.0] - 2022-09-07 

### Added
- Make BlockGasLimit configurable [#1418]
- Adapt to the new and improved reward strategy [#1320]
- Include BlockGenerator reward into the header [#1368]
- Search txs up to 10.000 blocks [#1401]
- Mempool updates its state at start-up automatically [#1258]
- Mempool discards any transaction with repeated nullifier [#1388]
- Detect any occurrences of missed fallback procedure [#1413] 
- Send reduction concurrently without blocking votes broadcast [#1442]

### Changed
- Extraction from mempool consider Gas expenditure estimation [#1421]
- Add GasLimit to the Block's header [#1416]
- Reduce wire messages needed for propagating an Agreement vote
- Improve fallback procedure to revert multiple ephemeral blocks [#1343]
- Disable provisioners records storing in api.db  [#1361]
- Optimize VST calls of a Provisioner per a consensus iteration [#1371]
- Call VerifyStateTransition in Reduction 2 only if it is a committee member [#1357]
- Drop topics.Block messages with invalid header.hash value [#1425]
- Request candidate block from arbitrary active nodes [#1359]
- Allow verifyFn to report errors correctly [#1436]
- Clone message.Agreement on notifying the consumer [#1433]
- Solve newblock msg errors [#1443]

### Security
- Ensure candidate block hash is equal to the BlockHash of the msg.header [#1364]
- Check quorum per each reduction step on agreement msg verification [#1373]
- Extend reduction 1 verification procedure with VST call [#1358]
- Verify block certificate prior to broadcast [#1446]
- Add support in sortition for bls public keys that starts with zeros [#1457]
- Discard aggrAgreement messages with invalid StepVotes [#1430]
- Check quorum target on certificate verification [#1432] 

### Fixed
- Default configuration values loading [#1419] 


## [0.5.0] - 2022-03-21 - Daybreak

### Added
- Populate graphql with transaction data [#1289]
- Decode transparent notes amount [#1296]
- Simple state persistence strategy [#1301]
- Recover contract state on node restart [#1313]
- Introduce Contract Storage revert in Fallback procedure [#1325]
- Test-harness multiple provisioners [#1318]
- Add `JSON` field to graphQL Transaction struct [#1329]
- Introduce DUSK_NETWORK_REUSE to instruct test-harness to reuse an existing state [#1340]

### Changed
- Disable magic error propagation [#1297]
- Adapt stake to new definitions [#1300]
- Change base dusk unit [#1305]
- Change FeePaid to GasSpent [#1311]
- Artificially increase block time [#1312]
- Remove Deadline context for Accept, Finalize and Persist rusk grpc calls [#1323]
- Mempool extraction by GasPrice [#1330]
- Retain transaction status [#1332]
- Throttle block acceptance on specified conditions [#1333]
- Add staking tx decoding tests [#1346]
- Throttle verifyFn in reduction_1 phase [#1342]
### Security
- Check transaction GasLimit against block GasLimit [#1316]
- Initialize/Verify candidate block timestamp properly [#1335]
- Introduce protocol version [#1337]
- Ensure that isValidBlock is called against chainTipBlk Seed [#1339]

### Fixed
- Fix reward and feesPaid in graphQL [#1315]
- Change GasSpent and GasLimit to float [#1345]

<!-- Issues -->

[#1481]: https://github.com/dusk-network/dusk-blockchain/issues/1481
[#1479]: https://github.com/dusk-network/dusk-blockchain/issues/1479
[#1464]: https://github.com/dusk-network/dusk-blockchain/issues/1464
[#1468]: https://github.com/dusk-network/dusk-blockchain/issues/1468
[#1457]: https://github.com/dusk-network/dusk-blockchain/issues/1457
[#1446]: https://github.com/dusk-network/dusk-blockchain/issues/1446
[#1443]: https://github.com/dusk-network/dusk-blockchain/issues/1443
[#1442]: https://github.com/dusk-network/dusk-blockchain/issues/1442
[#1436]: https://github.com/dusk-network/dusk-blockchain/issues/1436
[#1433]: https://github.com/dusk-network/dusk-blockchain/issues/1433
[#1432]: https://github.com/dusk-network/dusk-blockchain/issues/1432
[#1430]: https://github.com/dusk-network/dusk-blockchain/issues/1430
[#1425]: https://github.com/dusk-network/dusk-blockchain/issues/1425
[#1421]: https://github.com/dusk-network/dusk-blockchain/issues/1421
[#1419]: https://github.com/dusk-network/dusk-blockchain/issues/1419
[#1418]: https://github.com/dusk-network/dusk-blockchain/issues/1418
[#1416]: https://github.com/dusk-network/dusk-blockchain/issues/1416
[#1413]: https://github.com/dusk-network/dusk-blockchain/issues/1413
[#1401]: https://github.com/dusk-network/dusk-blockchain/issues/1401
[#1388]: https://github.com/dusk-network/dusk-blockchain/issues/1388
[#1373]: https://github.com/dusk-network/dusk-blockchain/issues/1373
[#1371]: https://github.com/dusk-network/dusk-blockchain/issues/1371
[#1368]: https://github.com/dusk-network/dusk-blockchain/issues/1368
[#1364]: https://github.com/dusk-network/dusk-blockchain/issues/1364
[#1361]: https://github.com/dusk-network/dusk-blockchain/issues/1361
[#1359]: https://github.com/dusk-network/dusk-blockchain/issues/1359
[#1358]: https://github.com/dusk-network/dusk-blockchain/issues/1358
[#1357]: https://github.com/dusk-network/dusk-blockchain/issues/1357
[#1343]: https://github.com/dusk-network/dusk-blockchain/issues/1343
[#1320]: https://github.com/dusk-network/dusk-blockchain/issues/1320
[#1258]: https://github.com/dusk-network/dusk-blockchain/issues/1258

<!-- PRs -->

[#1289]: https://github.com/dusk-network/dusk-blockchain/pull/1289 
[#1296]: https://github.com/dusk-network/dusk-blockchain/pull/1296
[#1297]: https://github.com/dusk-network/dusk-blockchain/pull/1297 
[#1300]: https://github.com/dusk-network/dusk-blockchain/pull/1300 
[#1301]: https://github.com/dusk-network/dusk-blockchain/pull/1301 
[#1305]: https://github.com/dusk-network/dusk-blockchain/pull/1305 
[#1311]: https://github.com/dusk-network/dusk-blockchain/pull/1311 
[#1315]: https://github.com/dusk-network/dusk-blockchain/pull/1315 
[#1316]: https://github.com/dusk-network/dusk-blockchain/pull/1316 
[#1312]: https://github.com/dusk-network/dusk-blockchain/pull/1312 
[#1313]: https://github.com/dusk-network/dusk-blockchain/pull/1313 
[#1323]: https://github.com/dusk-network/dusk-blockchain/pull/1323
[#1325]: https://github.com/dusk-network/dusk-blockchain/pull/1325 
[#1330]: https://github.com/dusk-network/dusk-blockchain/pull/1330 
[#1318]: https://github.com/dusk-network/dusk-blockchain/pull/1318 
[#1329]: https://github.com/dusk-network/dusk-blockchain/pull/1329 
[#1332]: https://github.com/dusk-network/dusk-blockchain/pull/1332
[#1333]: https://github.com/dusk-network/dusk-blockchain/pull/1333 
[#1335]: https://github.com/dusk-network/dusk-blockchain/pull/1335 
[#1337]: https://github.com/dusk-network/dusk-blockchain/pull/1337 
[#1339]: https://github.com/dusk-network/dusk-blockchain/pull/1339 
[#1340]: https://github.com/dusk-network/dusk-blockchain/pull/1340 
[#1342]: https://github.com/dusk-network/dusk-blockchain/pull/1342
[#1345]: https://github.com/dusk-network/dusk-blockchain/pull/1345 
[#1346]: https://github.com/dusk-network/dusk-blockchain/pull/1346 

<!-- Releases -->

[Unreleased]: https://github.com/dusk-network/dusk-blockchain/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/dusk-network/dusk-blockchain/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/dusk-network/dusk-blockchain/compare/v0.4.4...v0.5.0
