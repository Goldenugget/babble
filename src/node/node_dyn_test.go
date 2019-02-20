package node

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestMonologue(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peers := initPeers(1)
	nodes := initNodes(keys, peers, 100000, 1000, "inmem", 5*time.Millisecond, logger, t)
	//defer drawGraphs(nodes, t)

	target := 50
	err := gossip(nodes, target, true, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	checkGossip(nodes, 0, t)
}

func TestJoinRequest(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
		"monika",
	)
	newNode := newNode(peer, key, "new node", peerSet, 1000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer newNode.Shutdown()

	err = newNode.join()
	if err != nil {
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 30
	err = bombardAndWait(nodes, secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)
	checkPeerSets(nodes, t)
}

func TestLeaveRequest(t *testing.T) {
	n := 1
	f := func() {
		logger := common.NewTestLogger(t)
		keys, peerSet := initPeers(n)
		nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
		defer shutdownNodes(nodes)
		//defer drawGraphs(nodes, t)

		target := 30
		err := gossip(nodes, target, false, 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)

		leavingNode := nodes[n-1]

		err = leavingNode.Leave()
		if err != nil {
			t.Fatal(err)
		}

		if n == 1 {
			return
		}

		//Gossip some more
		secondTarget := target + 50
		err = bombardAndWait(nodes[0:n-1], secondTarget, 6*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes[0:n-1], 0, t)
		checkPeerSets(nodes[0:n-1], t)
	}

	for n <= 4 {
		f()
		n++
	}
}

func TestSuccessiveJoinRequest(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(1)

	node0 := newNode(peerSet.Peers[0], keys[0], "new node", peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
	defer node0.Shutdown()
	node0.RunAsync(true)

	nodes := []*Node{node0}
	//defer drawGraphs(nodes, t)

	target := 20
	for i := 1; i <= 3; i++ {
		peerSet := peers.NewPeerSet(node0.GetPeers())

		key, _ := crypto.GenerateECDSAKey()
		peer := peers.NewPeer(
			fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
			fmt.Sprintf("127.0.0.1:%d", 4240+i),
			"monika",
		)
		newNode := newNode(peer, key, "new node", peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)

		logger.Debugf("starting new node %d, %d", i, newNode.ID())
		defer newNode.Shutdown()
		newNode.RunAsync(true)

		nodes = append(nodes, newNode)

		//Gossip some more
		err := bombardAndWait(nodes, target, 10*time.Second)
		if err != nil {
			t.Fatal(err)
		}

		start := newNode.core.hg.FirstConsensusRound
		checkGossip(nodes, *start, t)

		target = target + 40
	}
}

func TestSuccessiveLeaveRequest(t *testing.T) {
	n := 4

	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(n)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)

	target := 0

	f := func() {
		//defer drawGraphs(nodes, t)
		target += 30
		err := gossip(nodes, target, false, 3*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)

		leavingNode := nodes[n-1]

		err = leavingNode.Leave()
		if err != nil {
			t.Fatal(err)
		}

		if n == 1 {
			return
		}

		nodes = nodes[0 : n-1]

		//Gossip some more
		target += 50
		err = bombardAndWait(nodes, target, 6*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		checkGossip(nodes, 0, t)
		checkPeerSets(nodes, t)
	}

	for n > 0 {
		f()
		n--
	}
}

func TestSimultaneusLeaveRequest(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	nodes := initNodes(keys, peerSet, 1000000, 1000, "inmem", 5*time.Millisecond, logger, t)
	defer shutdownNodes(nodes)
	//defer drawGraphs(nodes, t)

	target := 30
	err := gossip(nodes, target, false, 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes, 0, t)

	leavingNode := nodes[3]
	leavingNode2 := nodes[2]

	err = leavingNode.Leave()
	if err != nil {
		t.Fatal(err)
	}

	err = leavingNode2.Leave()
	if err != nil {
		t.Fatal(err)
	}

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes[0:2], secondTarget, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(nodes[0:2], 0, t)
	checkPeerSets(nodes[0:2], t)
}

func TestJoinFull(t *testing.T) {
	logger := common.NewTestLogger(t)
	keys, peerSet := initPeers(4)
	initialNodes := initNodes(keys, peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
	defer shutdownNodes(initialNodes)

	target := 30
	err := gossip(initialNodes, target, false, 6*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	checkGossip(initialNodes, 0, t)

	key, _ := crypto.GenerateECDSAKey()
	peer := peers.NewPeer(
		fmt.Sprintf("0x%X", crypto.FromECDSAPub(&key.PublicKey)),
		fmt.Sprint("127.0.0.1:4242"),
		"monika",
	)
	newNode := newNode(peer, key, "new node", peerSet, 1000000, 400, "inmem", 10*time.Millisecond, logger, t)
	defer newNode.Shutdown()

	//Run parallel routine to check newNode eventually reaches CatchingUp state.
	timeout := time.After(6 * time.Second)
	go func() {
		for {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for newNode to enter CatchingUp state")
			default:
			}
			if newNode.getState() == CatchingUp {
				break
			}
		}
	}()

	newNode.RunAsync(true)

	nodes := append(initialNodes, newNode)

	//defer drawGraphs(nodes, t)

	//Gossip some more
	secondTarget := target + 50
	err = bombardAndWait(nodes, secondTarget, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	start := newNode.core.hg.FirstConsensusRound
	checkGossip(nodes, *start, t)
	checkPeerSets(nodes, t)
}

func checkPeerSets(nodes []*Node, t *testing.T) {
	node0FP, err := nodes[0].core.hg.Store.GetAllPeerSets()
	if err != nil {
		t.Fatal(err)
	}
	for i := range nodes[1:] {
		nodeiFP, err := nodes[i].core.hg.Store.GetAllPeerSets()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(node0FP, nodeiFP) {
			t.Logf("Node 0 PeerSets: %v", node0FP)
			t.Logf("Node %d PeerSets: %v", i, nodeiFP)
			t.Fatalf("PeerSets defer")
		}
	}
}
