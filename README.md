![image](https://github.com/user-attachments/assets/230de78d-1283-4be6-94ee-2ee10e2d6b1b)


gosimnet
========

`gosimnet` is a compact network simulator. It is fairly minimal,
but still hopefully useful. It was written for testing with 
testing/synctest (which is very promising/exciting), but
can run without it too. gosimnet implements the `net.Conn`
interface to networks. The test file
https://github.com/glycerine/gosimnet/blob/master/simnet_test.go
illustrates its use.

I wrote this originally for https://github.com/glycerine/rpc25519 
system testing (simulating network chaos), without
a net.Conn interface. Then I realized it might 
be more broadly useful. So I pulled it out and condensed it, adding
a net.Conn interface, to make it potentially
usable for wider https://pkg.go.dev/testing/synctest testing.

Since synctest is in the experimental phase, 
getting early feedback in now to make it as
usable as possible would benefit all.

Quality: alpha. The net.Conn implementation in particular is
rough and ready. The scheduler and dispatcher
are pretty solid. They got pretty harsh testing
over in https://github.com/glycerine/rpc25519 .
The test here is just a minimal example. I have
not brought other more punishing tests over 
to keep this package petite and approachable, but that does
not reflect the true degree of testing that
has been done. Also I assume that with net.Conn
now available, almost anything can be made
to run on top of it, and so regular web servers
etc will provide coverage.

If you spot any issues, on any part, though,
please let me know.

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
