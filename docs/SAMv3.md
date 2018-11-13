# SAM 3.3

The SAM library is found in wire/sam3, and is a full Golang implementation based loosely on [this](https://bitbucket.org/kallevedin/sam3/overview)

The goal of this refactor is to make the codebase more easily maintanable, provide better code readability and make use of better practices to construct the library. Additionally, this library is extended to include functions for SAM version 3.3, allowing the use of `MASTER` sessions and subsessions on one single socket.

The SAM library is merely an API - it needs to be paired with an I2P router to work properly. For the moment, we can include the Java router in the codebase, but if the need arises later to create our own Golang version of an I2P router, that can be done, however as porting an I2P router is a pretty arduous task, for now it's best to hold off on that.

### Testing

To test this module, you should have an I2P router active on your system that can handle SAM 3.3 (currently, only the Java implementation supports up to 3.3). Make sure you turn on the SAM application bridge inside the router, and give it about 5-10 minutes to start up fully so that it will accept SAM messages and build tunnels for you. It is also advised that, if you are testing with a timeout, you should set this from anywhere between 2-5 minutes in case of extreme delay from the socket (happens mostly around startup, but messages can sometimes take 60 seconds to arrive from the SAM socket). Make sure your SAM bridge is on port `7656`, as that is where the testing files will try to connect to the router.