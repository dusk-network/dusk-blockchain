### RaptorQ Introduction

 RaptorQ codes are a new family of codes that provide superior
   flexibility, support for larger source block sizes, and better coding
   efficiency than Raptor codes in RFC 5053.  RaptorQ is also a fountain
   code, i.e., as many encoding symbols as needed can be generated on
   the fly by the encoder from the source symbols of a source block of
   data.  The decoder is able to recover the source block from almost
   any set of encoding symbols of sufficient cardinality -- in most
   cases, a set of cardinality equal to the number of source symbols is
   sufficient; in rare cases, a set of cardinality slightly more than
   the number of source symbols is required.

   Source: https://tools.ietf.org/html/rfc6330

### Implementation details

   RaptorQ package supports two utils - source object reader and writer. Both are based on 3rd C++ implementation of RaptorQ RFC6330 https://fenrirproject.org/Luker/libRaptorQ/. A Swig-based wrapper is pending to be added.

### Writer
Writer uses RaptorQ encoder to generate encoding symbols with size less than MTU len and write each of the encoding symbols to the UDP connection. 


### Reader
Reader represents an UDPReader that listens for UDP packets. For each source object on the wire, UDPReader instantiates a decoder. On receiving and encoding symbol, UDPReader passes it to the proper decoder. When a decoder is ready (the source object is fully received), it notifies the UDPReader with msg_id written in ReadySourceObjectChan.

Additionally, the schema of the data unit sent on the wire:

| source_object_id  | source_block_number | encoding_symbol_id | commonOTI | schemeSpecOTI | symbolData

### Development status
Until libRaptorQ is added in the repository CI and Swig-based golang wrapper is pushed, this package build is disabled.


