## start-node

This package compiles into a very simple executable which will start both the blind bid process, and the node process. The program expects both of these executables to be in the same directory as itself.

This executable should be mainly used for packaging with a release. Those building and running from source are encouraged to use the shell script in the `testnet` folder, as it saves the compilation and the moving of the `start-node` binary.
