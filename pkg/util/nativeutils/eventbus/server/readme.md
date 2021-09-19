# IPCBus server

This package contains the code required to allow the connection of an IPCBus 
client (or multiple clients) that will relay relevant messages across some 
form of read/write connection that can carry binary encoded messages.

As it is the controlling process, the constructor call for an IPCBus server 
includes the shell command to launch the client process, and includes the 
ability to start and stop the client process from the master (server), 
cascading shutdown signals this way for graceful shutdowns.
