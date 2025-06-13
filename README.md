![image](https://github.com/user-attachments/assets/d671bf05-5688-4f46-b685-63beb29826ab)

Image: simulacrum of the Go gopher happily toying with traffic.

gosimnet
========

gosimnet is a compact network simulator. It is minimal,
with a small code base and only two imports, to make
it useful for many test situations. It was written for 
testing with the new Go
testing/synctest facility, but
can run without it too. gosimnet implements the `net.Conn`
interface to networks. The test file
https://github.com/glycerine/gosimnet/blob/master/simnet_test.go
illustrates its use.

I wrote this originally for https://github.com/glycerine/rpc25519 
system testing (simulating network chaos), without
a net.Conn interface. Then I realized it might 
be more broadly useful. So I pulled it out and condensed it, adding
the net.Conn interface to allow testing regular Go networking
code. This hopefully will allow many to benefit
from https://pkg.go.dev/testing/synctest testing,
where talking to a real network is verboten.

The basic test here is just a minimal example. It
is meant to get you started.

The underlying rpc25519 package has an extensive 
test suite, and includes a full rsync-like protocol for
efficient filesystem sync. The underlying Simnet passes
all of those tests. They can be found in these files:

https://github.com/glycerine/rpc25519/blob/master/simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/jsync/rsync_simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/simgrid_test.go

Other limitations

Network connection endpoints ("addresses")
at the moment are kept as simple as 
possible -- just a string.

You can interpret this string as the host name,
as a host + port, or make it opaque if 
you wish by letting it be the
a string like "127.0.0.1:8080", or even
let it be a whole URL. The system
does not care what the endpoint string is,
only that each endpoint has a unique name.

Thus there is no need to emuate DNS to get
human readable addressing, and no network
specific address convention to emulate.
The convention of a server binding ":0" to get
a free port is not implemented (at the moment), as ports
are not really needed as a separate concept.

Servers and clients can be "multi-homed"
trivially, as their endpoint address is any 
string they choose.

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
