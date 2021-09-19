# IPCBus client

This package contains the code required to connect to an IPCBus and become a 
node on its message dispatch graph.

It is implemented by creating an in-process IPCBus master and negotiating 
subscriptions with the IPCBus server running this process. As such, the 
client runs a master broker and services in the client application bind to 
it to receive their messages and send their requests to the broker for 
relaying to the top level parent process. In this way, information can flow 
back and forth between multiple processes transparently and enable tighter, 
smaller module sizes and potential bugs hiding in complexity is minimized.
