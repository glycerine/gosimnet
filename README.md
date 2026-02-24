![image](https://github.com/user-attachments/assets/d671bf05-5688-4f46-b685-63beb29826ab)

Image: simulacrum of the Go gopher happily toying with traffic.

gosimnet
========

gosimnet is a compact network simulator.

gosimnet was written for testing with the new Go
testing/synctest facility, but
can run without it too. gosimnet implements the `net.Conn`
interface to networks. The test file
https://github.com/glycerine/gosimnet/blob/master/simnet_test.go
illustrates its use.

The basic test here is just a minimal example. It
is meant to get you started.

The underlying rpc25519 package has an extensive 
test suite, and includes a full rsync-like protocol for
efficient filesystem sync. The underlying Simnet passes
all of those tests. They can be found in these files:

https://github.com/glycerine/rpc25519/blob/master/simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/jsync/rsync_simnet_test.go

https://github.com/glycerine/rpc25519/blob/master/simgrid_test.go

Go version 1.25 or greater is required to access
the released testing/synctest package.

No special build tags are required now to 
access synctest.

# Seeding your Deterministic Simulation Testing (DST) run

Gosimnet supports deterministic replay for
Deterministic Simulation Testing (DST). The pseudo random
number generator seed for a run can be set as follows.

~~~
cfg := NewSimNetConfig()
cfg.InitialSimnetScenario = 43 
~~~

For rpc25519 setup activities that 
happen before the simnet is started,
we also have the rpc25519.PrepareForSimnet() 
facility which will seed the 
pseudo random number generator used by 
rpc25519.NewCallID() to obtain random
call and node identifiers.

~~~
package rpc25519

// PrepareForSimnet will re-seed our globalPRNG
// if int64seed is > 0. Otherwise we are a no-op,
// and altered will come back false. This is
// to avoid repeated output from the PRNG, since
// it was initialized with zero at init() time.
//
// This helps tube fuzz_test to get more deterministic
// NewCallID() responses from hdr.go.
func PrepareForSimnet(int64seed int64, allowRepeat bool) (altered bool) 
~~~

# network modeling API summary

You can use rpc25519.Config.GetSimnet() to get an *rpc.Simnet, and then...

(from go doc)

These pick out the net modeling stuff from the underlying rpc25519.
Documentation links:

https://pkg.go.dev/github.com/glycerine/rpc25519#Faultstate
https://pkg.go.dev/github.com/glycerine/rpc25519#Alteration
https://pkg.go.dev/github.com/glycerine/rpc25519#DropDeafSpec
https://pkg.go.dev/github.com/glycerine/rpc25519#Simnet
https://pkg.go.dev/github.com/glycerine/rpc25519#SimnetBatch
https://pkg.go.dev/github.com/glycerine/rpc25519#SimnetConnSummary
https://pkg.go.dev/github.com/glycerine/rpc25519#SimnetPeerStatus
https://pkg.go.dev/github.com/glycerine/rpc25519#SimnetSnapshot
https://pkg.go.dev/github.com/glycerine/rpc25519#SimnetSnapshotter


~~~
type Faultstate int

    Faultstate is one of HEALTHY, FAULTY, 
    ISOLATED, or FAULTY_ISOLATED.

    FAULTY models network card problems. 
    These can be either dropped sends or
    deaf reads. The probability of each 
    can be set independently. A card might
    be able to send messages but only read 
    incoming messages half the time,
    for example; an asymmetric and 
    intermittent failure mode.

    ISOLATED models a dead network card or
    switch port. 
    While this can be modelled more
    orthogonally as a collection of
    individual card faults on either side of
    switch (and may be implemented as 
    such internally) it is a common enough
    failure mode and test scenario that 
    giving it a distinct name enhances the
    API's usability and clarifies the 
    scenario being simulated.

    FAULTY_ISOLATED models network card 
    problems alongside network switch
    problems.

    HEALTHY means the network and card 
    are fully operational with respect to
    this circuit. This does not imply 
    that other circuits ending at the same
    host, or between the same pair of 
    hosts, are healthy too. A simhost 
    server will typically host many 
    circuit connections; at least 
    one per connected peer server.

    A circuit's powerOff status is 
    independent of its Faultstate, so that
    circuit faults like flakey network 
    cards and network isolatation (dead
    switches) survive (are not repaired 
    by) a simple host or circuit reboot.

    We reuse Faultstate for the whole 
    server state, to keep things simple 
    and to summarize the status of 
    all circuits therein. If a simnode 
    or Server is in powerOff, then all 
    circuits terminating there are also 
    in powerOff.

const (
    HEALTHY Faultstate = 0

    // more coarse than FAULTY. no communications 
    // with anyone else will happen.
    ISOLATED Faultstate = 1 

    // If a (deaf/drop) fault is applied 
    // to a HEALTHY circuit,
    // then the circuit is marked FAULTY.
    // If a repair removes the last fault, 
    // we change it back to HEALTHY.
    FAULTY Faultstate = 2 // some conn may drop sends, be deaf to reads

    // If a (deaf/drop) fault is 
    // applied to an ISOLATED circuit,
    // then the circuit is marked 
    // FAULTY_ISOLATED. If a repair removes 
    // the last fault, we change it 
    // back to ISOLATED.
    FAULTY_ISOLATED Faultstate = 3
)


type Alteration int 

    Alteration flags are used in 
    AlterCircuit() calls to specify 
    what change you want to a 
    specific network simnode.

const (
    UNDEFINED Alteration = 0
    SHUTDOWN  Alteration = 1
    POWERON   Alteration = 2
    ISOLATE   Alteration = 3
    UNISOLATE Alteration = 4
)


type DropDeafSpec struct {

    // false UpdateDeafReads means no change to deafRead
    // probability. The DeafReadsNewProb field is ignored.
    // This allows setting DeafReadsNewProb to 0 only
    // when you want to.
    UpdateDeafReads bool

    // probability of ignoring (being deaf) to a read.
    // 0 => never be deaf to a read (healthy).
    // 1 => ignore all reads (dead hardware).
    DeafReadsNewProb float64

    // false UpdateDropSends means the DropSendsNewProb
    // is ignored, and there is no change to the dropSend
    // probability.
    UpdateDropSends bool

    // probability of dropping a send.
    // 0 => never drop a send (healthy).
    // 1 => always drop a send (dead hardware).
    DropSendsNewProb float64
}

    DropDeafSpec specifies a network/netcard 
    fault with a given probability.



func (s *Simnet) AllHealthy(
    powerOnIfOff bool, 
    deliverDroppedSends bool,
    ) (err error)
    
    AllHealthy heals all partitions, and
    undoes all faults, 
    network wide. All circuits
    are returned to HEALTHY status. Their powerOff 
    status is not updated unless
    powerOnIfOff is also true. See also RepairSimnode 
    for single simnode repair.


func (s *Simnet) AlterCircuit(
     simnodeName string, 
     alter Alteration, 
     wholeHost bool,
     ) (undo Alteration, err error)

func (s *Simnet) AlterHost(
     simnodeName string, 
     alter Alteration,
     ) (undo Alteration, err error)
     
    we cannot guarantee that the undo will
    reverse all the changes if fine
    grained faults are in place; e.g. if only 
    one auto-cli was down and we
    shutdown the host, the undo of restart 
    will also bring up that auto-cli too.
    The undo is still very useful for tests 
    even without that guarantee.

func (s *Simnet) Close()

func (s *Simnet) FaultCircuit(
    origin, target string, 
    dd DropDeafSpec, 
    deliverDroppedSends bool,
    ) (err error)
    
    empty string target means all possible targets

func (s *Simnet) FaultHost(
     hostName string, 
     dd DropDeafSpec, 
     deliverDroppedSends bool,
     ) (err error)

func (s *Simnet) GetSimnetSnapshot() (snap *SimnetSnapshot)

func (s *Simnet) NewSimnetBatch(subwhen time.Time, subAsap bool) *SimnetBatch

func (s *Simnet) RepairCircuit(
     originName string, 
     unIsolate bool, 
     powerOnIfOff bool, 
     deliverDroppedSends bool,
     ) (err error)
     
    RepairCircuit restores the local 
    circuit to full working order.
    It undoes the effects of prior deafDrop 
    actions, if any. It does not change
    an isolated simnode's isolation unless 
    unIsolate is also true. See also
    RepairHost, AllHealthy.

func (s *Simnet) RepairHost(
     originName string, 
     unIsolate bool, 
     powerOnIfOff bool, 
     allHosts bool, 
     deliverDroppedSends bool,
     ) (err error)
     
    RepairHost repairs all the circuits on the host.


func (s *Simnet) SubmitBatch(batch *SimnetBatch)
    SubmitBatch does not block.

type SimnetBatch struct {
    // Has unexported fields.
}
    SimnetBatch is a proposed design for 
    sending in a batch of network
    fault/repair/config changes at once. 
    Currently a prototype; not really
    finished/tested yet.

func (b *SimnetBatch) AllHealthy(
    powerOnIfOff bool, 
    deliverDroppedSends bool)

func (b *SimnetBatch) AlterCircuit(
    simnodeName string, 
    alter Alteration, 
    wholeHost bool,
    )

func (b *SimnetBatch) AlterHost(
     simnodeName string, 
     alter Alteration,
     )

    we cannot guarantee that the undo 
    will reverse all the changes if fine
    grained faults are in place; e.g. if 
    only one auto-cli was down and we
    shutdown the host, the undo of restart 
    will also bring up that auto-cli too.
    The undo is still very useful for tests 
    even without that guarantee.

func (b *SimnetBatch) FaultCircuit(
     origin string, 
     target string, 
     dd DropDeafSpec, 
     deliverDroppedSends bool,
     )
     
    empty string target means all possible targets

func (b *SimnetBatch) FaultHost(
    hostName string, 
    dd DropDeafSpec, 
    deliverDroppedSends bool,
    )

func (b *SimnetBatch) GetSimnetSnapshot()

func (b *SimnetBatch) RepairCircuit(
     originName string, 
     unIsolate bool, 
     powerOnIfOff bool, 
     deliverDroppedSends bool,
     )

func (b *SimnetBatch) RepairHost(
     originName string, 
     unIsolate bool, 
     powerOnIfOff bool, 
     allHosts bool, 
     deliverDroppedSends bool,
     )
     
    RepairHost repairs all the circuits on the host.

type SimnetConnSummary struct {
    OriginIsCli      bool
    Origin           string
    OriginState      Faultstate
    OriginConnClosed bool
    OriginPoweroff   bool
    Target           string
    TargetState      Faultstate
    TargetConnClosed bool
    TargetPoweroff   bool
    DropSendProb     float64
    DeafReadProb     float64

    // origin Q summary
    Qs string

    // origin priority queues:
    // Qs is the convenient/already stringified form of
    // these origin queues.
    // These allow stronger test assertions.  They are deep clones
    // and so mostly race free except for the
    // pointers mop.{origin,target,origTimerMop,msg,sendmop,readmop},
    // access those only after the simnet has been shutdown.
    // The proceed channel is always nil.
    DroppedSendQ *pq
    DeafReadQ    *pq
    ReadQ        *pq
    PreArrQ      *pq
    TimerQ       *pq
}

    SimnetConnSummary is part 
    of a SimnetSnapshot. 
    It conveys the full state of a
    net.Conn connection endpoint.

func (z *SimnetConnSummary) String() (r string)

type SimnetPeerStatus struct {
    Name         string
    Conn         []*SimnetConnSummary
    Connmap      map[string]*SimnetConnSummary
    ServerState  Faultstate
    Poweroff     bool
    LC           int64
    ServerBaseID string
    IsLoneCli    bool // and not really a peer server with auto-cli
}

    SimnetPeerStatus is a part of a SimnetSnapshot.

func (z *SimnetPeerStatus) String() (r string)

type SimnetSnapshot struct {
    Asof               time.Time
    Loopi              int64
    NetClosed          bool
    GetSimnetStatusErr error
    Cfg                SimNetConfig
    PeerConnCount      int
    LoneCliConnCount   int

    // mop creation/finish data.
    Xcountsn  int64       // number of mop issued
    Xfinorder []int64     // finish order (nextMopSn at time of finish)
    Xwhence   []string    // file:line creation place
    Xkind     []mopkind   // send,read,timer,discard,...
    Xissuetm  []time.Time // when issued
    Xfintm    []time.Time // when finished

    Xhash string // hash of the sequence

    ScenarioNum    int
    ScenarioSeed   [32]byte
    ScenarioTick   time.Duration
    ScenarioMinHop time.Duration
    ScenarioMaxHop time.Duration

    Peer    []*SimnetPeerStatus
    Peermap map[string]*SimnetPeerStatus
    LoneCli map[string]*SimnetPeerStatus // not really a peer but meh.

    // Has unexported fields.
}

    SimnetSnapshot is returned by
    SimnetSnapshotter.GetSimnetSnapshot(),
    and gives a picture of the
    network at a moment in time.

func (z *SimnetSnapshot) LongString() (r string)

    LongString provides all the details 
    even when the network is all healthy.

func (z *SimnetSnapshot) ShortString() (r string)
    ShortString: if everything is healthy, just give a short summary. Otherwise
    give the full snapshot.

func (z *SimnetSnapshot) String() (r string)
    String: if everything is healthy, 
    just give a short summary. Otherwise give
    the full snapshot.

func (snap *SimnetSnapshot) ToFile(nm string)

type SimnetSnapshotter struct {
    // Has unexported fields.
}

func (s *SimnetSnapshotter) GetSimnetSnapshot() *SimnetSnapshot
~~~

# naming -- suprisingly simple

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

You must ensure this uniqueness across all
network nodes. Beyond that, the name can
be anything that makes your modeling easier.

Thus there is no need to emuate DNS to get
human readable addressing, and no network
specific address convention to emulate.
The convention of a server binding ":0" to get
a free port is not implemented (at the moment), as ports
are not really needed as a separate concept.

Servers and clients can be "multi-homed"
trivially, as their endpoint address is any 
string you choose; so long as they are
unique within a given simnet.

For convenience, the rpc25519.NewCallID(shortName) 
call can be used to generate a long (but 
not cryptographically) random 21 byte ID that will 
be unique with very high probability.

---
Author: Jason E. Aten, Ph.D.

License: 3-clause BSD, same as Go. See the LICENSE file.
