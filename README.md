![image](https://github.com/user-attachments/assets/d671bf05-5688-4f46-b685-63beb29826ab)

image: simulacrum of the Go gopher happily toying with traffic.

gosimnet
========

`gosimnet` is a compact network simulator. It is minimal,
with a small code base and only two imports, to make
it useful for many test situations. It was written for testing with 
testing/synctest (which is very promising/exciting), but
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

(Since synctest is in the experimental phase, 
getting early feedback in now to make it as
usable as possible would benefit the Go community).

gosimnet quality: high; ready for stress testing. 
I'm not aware of any issues, but this needs 
additional real use. Bring on your most punishing tests;
and please file any issues that you encounter, or
suggestions for improvement.

The basic test here is just a minimal example. 
This short demo does not reflect the true 
degree of testing that has been done.
I have not brought over other more 
punishing tests, to keep this package petite 
and approachable. This also avoids some of
the extensive setup required that would
arguably distract from understanding the simple
facility provided here, but I may do so in the future.

My rpc25519 package has an extensive test suite, and
includes a full rsync-like protocol for
efficient filesystem sync. gosimnet passes
all of those tests. They can be found in these two files:

https://github.com/glycerine/rpc25519/blob/master/simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/jsync/rsync_simnet_test.go

The caveat to this is that the net.Conn
interface was layered on top of the core
simnet functionality for pushing and
pulling bytes through the network. So
although the network simulation is
very solid, the net.Conn layer on
top is still fairly new. As such, it 
has received much less exercise.
It could use review/other eyes
on its net.Conn.Read() and Write()
methods to catch or anticipate
cases that are not yet handled, and
more tests. 

https://github.com/glycerine/gosimnet/blob/master/simnet_server.go#L183

https://github.com/glycerine/gosimnet/blob/master/simnet_server.go#L311

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
This would be trivial to add, but just
adds extra steps to server setup. All
this machinery is a vestige of the 16-bit computers that 
the Internet Protocol was designed on,
and can be omitted.

Servers and clients can be easily "multi-homed",
as their origin address is any 
string they choose.

A small, conceptual, disadvantage of this simple 
string-as-address design (skipping DNS and ports) might be that
there is no guard against a user accidentally simulating
their host being in two physical
places at once by mistake. Rules to enforce
conventions about which endpoint has which
string can be easily added if their
need becomes apparent.

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
