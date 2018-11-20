# SAM 3.3

The SAM library is found in wire/sam3, and is a full Golang implementation based loosely on [this](https://bitbucket.org/kallevedin/sam3/overview). The full specifications of SAM can be found [here](https://geti2p.net/en/docs/api/samv3).

The goal of this refactor is to make the codebase more easily maintanable, provide better code readability and make use of better practices to construct the library. Additionally, this library is extended to include functions for SAM version 3.3, allowing the use of `MASTER` sessions and subsessions on one single socket.

The SAM library is merely an API - it needs to be paired with an I2P router to work properly. For the moment, we can include the Java router in the codebase, but if the need arises later to create our own Golang version of an I2P router, that can be done, however as porting an I2P router is a pretty arduous task, for now it's best to hold off on that.

## Testing

To test this module, you should have an I2P router active on your system that can handle SAM 3.3 (currently, only the [Java implementation](https://geti2p.net/en/download) supports up to 3.3). Make sure you turn on the SAM application bridge inside the router, and give it about 5-10 minutes to start up fully so that it will accept SAM messages and build tunnels for you (as a general rule, you can consider it safe to start testing once the 'Rejecting tunnels: Starting up' message has disappeared, and 'Accepting tunnels' has taken it's place). It is also advised that, if you are testing with a timeout, you should set this from anywhere between 2-5 minutes in case of extreme delay from the socket (happens mostly around startup, but messages can sometimes take 60 seconds to arrive). Make sure your SAM bridge is on port `7656`, as that is where the testing files will try to connect to the router (it should automatically bind to that port).

**A note about forwarding**

The SAM streaming library has a possibility to forward incoming streams to another destination. However, testing this out on a single SAM bridge proves to be quite difficult - hence why there are no tests included for this. An attempt at creating 3 streaming sessions and trying to set one up for forwarding will result in connection resets and throw tests off. This function is better tested with multiple devices, for which I'll most likely include code later.

## Usage

The SAM library has a few basic lines of code for setup if you want to use it. 

**Error handling will be omitted for clarity in this documentation.**

```go
// Open connection with the SAM bridge
sam, err := NewSAM("127.0.0.1:7656")

// Get an address
keys, err := sam.NewKeys()

defer sam.Close()
```

This sets you up with a connection to the router's SAM bridge and gives you a set of keys to start a session with. Make sure you add `defer sam.Close()` whenever you open up a connection with the router, to make sure everything will properly exit after you're done. There are three options available:

### Streaming sessions

Streaming sessions allow the continuous streaming of data between two peers until one of them closes the socket.

A streaming session can be opened on the socket by using this code:

```go
stream, err := sam.NewStreamSession("stream", keys, []string{}, mediumShuffle)
```

If successful, `stream` will contain a struct with information and methods needed to make use of the streaming session. Let's take a look at the `StreamSession` struct.

```go
type StreamSession struct {
	ID      string        // Tunnel name
	Keys    I2PKeys       // I2P destination keys
	Conn    net.Conn      // Connection to SAM bridge
	Streams []*StreamConn // All open connections in the StreamSession go here
}
```

Most of the fields here are self-explanatory. The `Streams` field will contain all open connections currently on the session, and will be empty on startup. Connections can then be made with three different methods.

**CONNECT**

This method will request a connection from a peer on the network to open a streaming connection.

```go
func (ss *StreamSession) Connect(addr string) (*StreamConn, error) {
	// Get new SAM socket for the StreamConn
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	// Send message to SAM
	msg := []byte("STREAM CONNECT ID=" + ss.ID + " DESTINATION=" + addr + " SILENT=false\n")
	text, err := SendToBridge(msg, s.Conn) // will return once the counterparty accepts or rejects
	if err != nil {
		s.Close()
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		s.Close()
		return nil, err
	}

	sc := StreamConn{
		Session:    ss,
		LocalAddr:  ss.Keys.Addr,
		RemoteAddr: addr,
		Conn:       s.Conn,
	}

	ss.Streams = append(ss.Streams, &sc)
	return &sc, nil
}
```

As per the SAMv3 specification, the function will open another socket connection to host the stream on, and will then send the `STREAM CONNECT` message to the bridge. The bridge will then attempt to establish a streaming connection with `addr`. The response will be handled by the `HandleReponse` method, and if successful, it will return a pointer to a `StreamConn`, which contains information and methods for reading from and writing to a stream. This pointer will also be added to the `Streams` field of the `StreamSession`. 

**ACCEPT**

To open a socket and start accepting incoming requests, the `Accept` method can be called.

```go
func (ss *StreamSession) Accept(silent bool) (*StreamConn, error) {
	// Get new SAM socket for the StreamConn
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	// Send message to SAM
	msg := []byte("STREAM ACCEPT ID=" + ss.ID + " SILENT=" +
		strconv.FormatBool(silent) + "\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		return nil, err
	}

	var sc StreamConn

	// If SILENT=false, wait for the message containing the dest
	if !silent {
		// Read message once we get one
		buf := make([]byte, acceptMessageLen)
		if _, err := s.Conn.Read(buf); err != nil {
			return nil, err
		}

		// Handle whatever comes through the socket
		i := bytes.IndexByte(buf, '\n')
		if i == -1 {
			return nil, errors.New("SAM bridge message was not properly terminated")
		}

		data := string(buf[:i])
		dest := strings.Split(data, "FROM_PORT=")[0]

		// Create StreamConn
		sc = StreamConn{
			Session:    ss,
			Conn:       s.Conn,
			LocalAddr:  ss.Keys.Addr,
			RemoteAddr: dest,
		}
	} else {
		// If SILENT=true, just return a connection
		sc = StreamConn{
			Session:   ss,
			Conn:      s.Conn,
			LocalAddr: ss.Keys.Addr,
		}
	}

	// Add connection to the session list and return it
	ss.Streams = append(ss.Streams, &sc)
	return &sc, nil
}
```

This works much like `Connect`, but instead it writes a `STREAM ACCEPT` message to the socket. If no errors are returned from the socket, the SAM bridge will start waiting for connection requests, and accept the first incoming one. If the parameter `silent` was set to false, the client will wait for the bridge to write something back. This message will contain the destination that the stream is coming from, as well as the `FROM_PORT` and `TO_PORT`. Upon receiving this message, the function will populate a `StreamConn` struct with the information it received and return a pointer to it. If `silent` was set to true, the bridge will not return a message containing counterparty information and the client will just return a `StreamConn` without a `RemoteAddr` field. The `StreamConn` will also get added to the `Streams` array.

**FORWARD**

SAM also offers the possibility to open a forwarding socket to re-direct connection requests.

```go
func (ss *StreamSession) Forward(addr string, port string) (net.Conn, error) {
	// Get new SAM socket for the stream
	s, err := NewSAM(ss.Conn.RemoteAddr().String())

	// Send message to SAM
	msg := []byte("STREAM FORWARD ID=" + ss.ID + " PORT=" + port + " HOST=" + addr + " SILENT=false\n")
	text, err := SendToBridge(msg, s.Conn)
	if err != nil {
		return nil, err
	}

	if err := ss.HandleResponse(text); err != nil {
		return nil, err
	}

	return s.Conn, nil
}
```

This also works much like the previous functions, but instead sends a `STREAM FORWARD` command to the socket. If no errors are returned, the function will simply return a connection, which can be closed once the user wishes to stop forwarding connections. The SAM bridge will handle all the redirection, so on the user's end, not much else is to be done here.

Let's now take a look at the `StreamConn` struct.

```go
type StreamConn struct {
	Session    *StreamSession // The StreamSession this connection is running on.
	LocalAddr  string         // Our destination
	RemoteAddr string         // Destination of the streaming counterparty
	Conn       net.Conn       // Connection to the SAM socket
}
```

A `StreamConn` will contain a pointer back to the session it belongs to, as well as the local and remote address of the stream connection. At the bottom, a `net.Conn` is kept on which the reading and writing occurs.

To write to a stream, use the `Write` method:

```go
func (sc *StreamConn) Write(buf []byte) (int, error) {
	n, err := sc.Conn.Write(buf)
	return n, err
}
```

Quite straightforward, the function will return the amount of bytes written, and an error if applicable.

To read from a stream, use the `Read` method:

```go
func (sc *StreamConn) Read(buf []byte) (int, error) {
	n, err := sc.Conn.Read(buf)
	return n, err
}
```

Essentially identical to `Write` except for in the function call.

When done streaming, the `Close` method can be called on the `StreamConn`:

```go
func (sc *StreamConn) Close() error {
	// Remove itself from the Streams array
	ss := sc.Session.Streams
	for i, conn := range ss {
		if conn == sc {
			ss[i] = nil
			if i != len(ss)-1 {
				ss[i], ss[len(ss)-1] = ss[len(ss)-1], ss[i]
			}

			ss = ss[:len(ss)-1]
			sc.Session.Streams = ss
		}
	}

	WriteMessage([]byte("EXIT"), sc.Conn)
	return sc.Conn.Close()
}
```

The `StreamConn` will first remove itself from the `Streams` array of the corresponding `StreamSession`. It will then write the `EXIT` command to the SAM bridge, and close to connection to the previously opened socket.

Finally, to close a `StreamSession` simply call the `Close` method:

```go
func (ss *StreamSession) Close() error {
	// Close all connections first. Since StreamConns reorganize the array
	// themselves, keep closing the StreamConn on the zero index
	// until a nil pointer is encountered.
	for {
		if len(ss.Streams) == 0 {
			break
		}

		if err := ss.Streams[0].Close(); err != nil {
			return err
		}
	}

	WriteMessage([]byte("EXIT"), ss.Conn)
	return ss.Conn.Close()
}
```

This will make sure all `StreamConn` instances are closed first, and it will then write the `EXIT` command to the socket. This will close the connection with the SAM bridge. To start a new session, you will first have to open a new socket connection with `NewSAM`. **NOTE: If you have called `defer sam.Close()`, the call to `stream.Close()` will not be necessary, as the `sam` object will properly cancel your session on close.**

### Datagram sessions

Datagram sessions allow for the sending and receiving of repliable datagrams over I2P. To start a datagram session, use the following code:

```go
dg, err := sam.NewDatagramSession("dg", keys, []string{}, mediumShuffle)
```

This will return a `DatagramSession` object, and an error if applicable. Let's take a look at the `DatagramSession` object.

```go
type DatagramSession struct {
	ID       string       // Session name
	Keys     I2PKeys      // I2P keys
	Conn     net.Conn     // Connection to the SAM socket
	UDPConn  *net.UDPConn // Used to deliver datagrams
	RUDPAddr *net.UDPAddr // The SAM socket UDP address
	FromPort string       // FROM_PORT specified on creation
	ToPort   string       // TO_PORT specified on creation
}
```

The session will contain all basic information: ID, keys, a connection to the bridge. Additionally, it will contain a UDP connection and a UDP address to send and receive datagrams with. Lastly, the `FROM_PORT` and `TO_PORT` will be saved as well, which can be specified in the `SAMOpt` parameter in `NewDatagramSession` (which is passed `[]string{}` in the above function - they will be 0 as per the standard).

To send datagrams, use the `Write` method:

```go
func (s *DatagramSession) Write(b []byte, addr string) (int, error) {
	header := []byte("3.3 " + s.ID + " " + addr + " FROM_PORT=" + s.FromPort + " TO_PORT=" + s.ToPort + "\n")
	msg := append(header, b...)
	n, err := s.UDPConn.WriteToUDP(msg, s.RUDPAddr)

	return n, err
}
```

This will send a repliable datagram to `addr`. The function will return the number of bytes written, and an error if applicable.

To receive datagrams, use the `Read` method:

```go
func (s *DatagramSession) Read() ([]byte, string, error) {
	var dest string
	buf := make([]byte, 32768+4168) // Max datagram size + max SAM bridge message size.
	n, sAddr, err := s.UDPConn.ReadFromUDP(buf)
	if err != nil {
		return nil, dest, err
	}

	// Only accept incoming UDP messages from the SAM socket we're connected to.
	if !sAddr.IP.Equal(s.RUDPAddr.IP) {
		return nil, dest, fmt.Errorf("datagram received from wrong address: expected %v, actual %v",
			s.RUDPAddr.IP, sAddr.IP)
	}

	// Split message lines first
	i := bytes.IndexByte(buf, byte('\n'))
	msg, data := string(buf[:i]), buf[i+1:n]

	// Split message into fields
	fields := strings.Split(msg, " ")

	// Handle message
	for _, field := range fields {
		switch {
		case strings.Contains(field, "DESTINATION="):
			dest = strings.TrimPrefix(field, "DESTINATION=")
		default:
			continue
		}
	}

	return data, dest, nil
}
```

This will read from the `UDPConn` on the `DatagramSession`, which is where datagrams will be delivered. It will split the header data and the message, and return the message, the destination it was sent from, and an error if applicable.

To close the datagram session, simply call the `Close` method:

```go
func (s *DatagramSession) Close() error {
	WriteMessage([]byte("EXIT"), s.Conn)
	if err := s.Conn.Close(); err != nil {
		return err
	}

	err := s.UDPConn.Close()
	return err
}
```

This will write the `EXIT` command to the bridge and close the connection with the SAM bridge, ending the session. If you wish to start a new session after this, you will have to open a new connection to the SAM bridge with `NewSAM`. **NOTE: If you have called `defer sam.Close()`, the call to `dg.Close()` will not be necessary, as the `sam` object will properly cancel your session on close.**

### Raw sessions

Much like the datagram session, it's used for sending and receiving non-repliable datagrams. To start a raw session, use the following code:

```go
raw, err := sam.NewRawSession("raw", keys, []string{}, mediumShuffle)
```

This will return a `RawSession` struct, and an error if applicable. Let's look at the `RawSession` struct.

```go
type RawSession struct {
	ID       string       // Session name
	Keys     I2PKeys      // I2P keys
	Conn     net.Conn     // Connection to the SAM socket
	UDPConn  *net.UDPConn // Used to deliver and read datagrams
	RUDPAddr *net.UDPAddr // The SAM socket UDP address
	FromPort string       // FROM_PORT specified on creation
	ToPort   string       // TO_PORT specified on creation
	Protocol string       // PROTOCOL specified on creation
}
```

Like a datagram session, it contains the ID, keys, connection to the bridge, as well as a UDP conection and a UDP address for sending and receiving non-repliable datagrams. The ports are included as well, and additionally, a raw session can also define a `PROTOCOL` on setup, passed through `SAMOpt`.

To send non-repliable datagrams, use the `Write` method:

```go
func (s *RawSession) Write(b []byte, addr string) (n int, err error) {
	header := []byte("3.3 " + s.ID + " " + addr + " FROM_PORT=" + s.FromPort + " TO_PORT=" + s.ToPort +
		" PROTOCOL=" + s.Protocol + "\n")
	msg := append(header, b...)
	n, err = s.UDPConn.WriteToUDP(msg, s.RUDPAddr)

	return n, err
}
```

Nearly identical to the `Write` method on `DatagramSession`, although a protocol number is also included in the message. Return the amount of bytes written, and an error if applicable.

To receive non-repliable datagrams, use the `Read` method:

```go
func (s *RawSession) Read() ([]byte, error) {
	buf := make([]byte, 32768+67) // Max datagram size + max SAM bridge message size
	n, sAddr, err := s.UDPConn.ReadFromUDP(buf)
	if err != nil {
		return nil, err
	}

	// Only accept incoming UDP messages from the SAM socket we're connected to.
	if !sAddr.IP.Equal(s.RUDPAddr.IP) {
		return nil, fmt.Errorf("datagram received from wrong address: expected %v, actual %v",
			s.RUDPAddr.IP, sAddr.IP)
	}

	// If a header is included, split message lines first
	i := bytes.IndexByte(buf, byte('\n'))
	var data []byte
	if i != -1 {
		_, data = string(buf[:i]), buf[i+1:n]
	} else {
		data = buf[:n]
	}

	return data, nil
}
```

Again, much like the `Read` method on `DatagramSession`, but with much less header handling. Raw sessions do allow the inclusion of headers in the datagram, but they do not include the destination, and any further information is negligible, so it is discarded (and unless specified on session creation, non-repliable datagrams will never include a header).

Finally, to close a `RawSession`, simply call the `Close` method:

```go
func (s *RawSession) Close() error {
	WriteMessage([]byte("EXIT"), s.Conn)
	if err := s.Conn.Close(); err != nil {
		return err
	}

	err := s.UDPConn.Close()
	return err
}
```

This will write the `EXIT` command to the socket, and close the connection with the bridge. If you wish to open a new session, you will have to first establish a new connection to the bridge with the `NewSAM` function. **NOTE: If you have called `defer sam.Close()`, the call to `raw.Close()` will not be necessary, as the `sam` object will properly cancel your session on close.**

### Master sessions

A master session allows you to host any of the other sessions as subsession on the same socket connection, allowing you to have one I2P address for all your incoming and outgoing data transmissions, and preventing the socket connection from closing if you end a subsession.

To set up a master session after obtaining a socket connection and a pair of keys, use the following code:

```go
master, err := sam.NewMasterSession("master", keys, mediumShuffle)
```

By calling the `NewMasterSession` method on the `sam` object we got earlier, and using our `keys`, we can open a master session on the SAM bridge. The `mediumShuffle` parameter is a preset slice of options to pass to the router (all such options can be found in the `presets.go` file). Let's take a look at the `MasterSession` struct:

```go
type MasterSession struct {
	ID       string             // Session ID
	Keys     I2PKeys            // I2P keys
	Conn     net.Conn           // Connection to the SAM bridge
	Sessions map[string]Session // Maps session IDs to their structs for easy access
	SIDs     []string           // All sessions on the master session by ID
}
```

The master session will contain it's ID, keys, a connection to the socket, as well as a map of IDs linked to their respective subsessions, and an array of all subsession IDs running on this master session.

The `MasterSession` then allows you to add subsessions on the socket. There are three types of subsessions you can open:
* Streaming sessions - `master.AddStream("stream", []string{})`
* Datagram sessions - `master.AddDatagram("dg", []string{})`
* Raw sessions - `master.AddRaw("raw", []string{})`

The functions will return their respective sessions, as well as an error object, in case an error is encountered. You can then use these sessions as you would normally.

To close a subsession, call the `Remove` method on the `MasterSession`, passing the ID of the subsession. An example of closing a subsession:

```go
master.Remove("stream")
```

This will remove the subsession with the ID "stream" from the master session.

Finally, to close a master session, call the `Close` method:

```go
func (s *MasterSession) Close() error {
	if len(s.SIDs) > 0 {
		// Close all subsessions first
		for {
			// Keep deleting on zero index until the slice is empty
			if len(s.SIDs) == 0 {
				break
			}

			if err := s.Remove(s.SIDs[0]); err != nil {
				WriteMessage([]byte("EXIT"), s.Conn)
				s.Conn.Close()
				return err
			}
		}
	}

	// Close connection to SAM bridge.
	WriteMessage([]byte("EXIT"), s.Conn)
	err := s.Conn.Close()
	return err
}
```

This will first make sure all subsessions are closed accordingly, before writing the `EXIT` command to the socket, and closing the connection. To start a new session, you will have to get a new socket connection with `NewSAM`. **NOTE: If you have called `defer sam.Close()`, the call to `master.Close()` will not be necessary, as the `sam` object will properly cancel your session on close.**
