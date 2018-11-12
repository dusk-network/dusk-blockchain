# SAM 3.3

The SAM library is found in wire/sam3, and is a full Golang implementation based loosely on [this](https://bitbucket.org/kallevedin/sam3/overview)

The goal of this refactor is to make the codebase more easily maintanable, provide better code readability and make use of better practices to construct the library. Additionally, this library is extended to include functions for SAM version 3.3, allowing the use of `MASTER` sessions and subsessions on one single socket.

The SAM library is merely an API - it needs to be paired with an I2P router to work properly. For the moment, we can include the Java router in the codebase, but if the need arises later to create our own Golang version of an I2P router, that can be done, however as porting an I2P router is a pretty arduous task, for now it's best to hold off on that.

### Testing

To test this module, you should have an I2P router active on your system that can handle SAM 3.3 (currently, only the Java implementation supports up to 3.3). The testing files will be written out later.