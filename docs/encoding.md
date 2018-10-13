# [Encoding]

The encoding library will provide methods for all basic data types to be serialized and deserialized.
Types are divided into different files, with types of a predictable size employing a free list of buffers
to do their encoding/decoding operations with.

As the library gets written out, this documentation will expand and become more in-depth to provide a clear
example of how the library works and how it can be implemented in the rest of the software.