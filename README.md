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
I have not brought over (yet) other more 
punishing tests, to keep this package petite 
and approachable.

My rpc25519 package has an extensive test suite, and
includes a full rsync-like protocol for
efficient filesystem sync. gosimnet passes
all of those tests. They can be found in these two files:

https://github.com/glycerine/rpc25519/blob/master/simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/jsync/rsync_simnet_test.go

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
