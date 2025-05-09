gosimnet
========

`gosimnet` is a simulated network for testing with 
testing/synctest. gosimnet implements the `net.Conn`
interface to networks. The test file
https://github.com/glycerine/gosimnet/blob/master/simnet_test.go
illustrates its use.

I wrote this originally for https://github.com/glycerine/rpc25519 
system testing (simulating network chaos), without
a net.Conn interface. Then I realized it might 
be more broadly useful. So I pulled it out and condensed it, adding
a net.Conn interface, to make it potentially
usable for the https://pkg.go.dev/testing/synctest
Since synctest is in the experimental phase, 
getting early feedback in now to make it as
usable as possible would benefit all.

The net.Conn implementation in particular is
rough and ready. If you spot any issues, 
please let me know.

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
