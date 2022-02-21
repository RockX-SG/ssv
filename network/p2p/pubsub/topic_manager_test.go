package pubsub

import (
	"context"
	"fmt"
	"github.com/bloxapp/ssv/network/p2p/discovery"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTopicManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nPeers := 8
	nTopics := 16

	peers := newPeers(ctx, t, nPeers)

	subTopic := func(p *P, i int, potentialErrs ...error) {
		tname := topicName(i)
		in, err := p.tm.Subscribe(tname)
		if len(potentialErrs) == 0 {
			require.NoError(t, err)
		} else if err != nil {
			found := false
			for _, e := range potentialErrs {
				if e.Error() == err.Error() {
					found = true
					break
				}
			}
			require.True(t, found)
		}
		if in == nil {
			return
		}
		for ctx.Err() == nil {
			next := <-in
			p.saveMsg(tname, next)
		}
	}

	// listen to topics
	for i := 0; i < nTopics; i++ {
		for _, p := range peers {
			go subTopic(p, i)
			// simulate concurrency, by trying to subscribe twice
			<-time.After(time.Millisecond)
			go subTopic(p, i, ErrInProcess, errTopicAlreadyExists)
		}
	}

	// let the peers join topics
	<-time.After(time.Second * 5)

	// publish some messages
	for i := 0; i < nTopics; i++ {
		for _, p := range peers {
			go func(p *P, i int) {
				require.NoError(t, p.tm.Broadcast(topicName(i), []byte("dummy message 1"), time.Second*3))
			}(p, i)
		}
	}

	// let the messages propagate
	<-time.After(time.Second * 5)

	// check number of topics
	for _, p := range peers {
		require.Len(t, p.tm.Topics(), nTopics)
	}

	// check number of peers and messages
	for i := 0; i < nTopics; i++ {
		for _, p := range peers {
			peers, err := p.tm.Peers(topicName(i))
			require.NoError(t, err)
			require.Len(t, peers, nPeers-1)
			c := p.getCount(topicName(i))
			//t.Logf("peer %d got %d messages for %s", j, c, topicName(i))
			require.Greater(t, float64(c), float64(nPeers)*0.5)
			require.Less(t, float64(c), float64(nPeers)*1.2)
		}
	}

	// unsubscribe
	var wg sync.WaitGroup
	for i := 0; i < nTopics; i++ {
		for _, p := range peers {
			wg.Add(1)
			go func(p *P, i int) {
				defer wg.Done()
				require.NoError(t, p.tm.Unsubscribe(topicName(i)))
			}(p, i)
		}
	}
	wg.Wait()
}

func topicName(i int) string {
	return fmt.Sprintf("ssv-test-%d", i)
}

type P struct {
	host host.Host
	ps   *pubsub.PubSub
	tm   *topicManager

	connsCount uint64

	msgsLock sync.Locker
	msgs     map[string][]*pubsub.Message
}

func (p *P) getCount(t string) int {
	p.msgsLock.Lock()
	defer p.msgsLock.Unlock()

	msgs, ok := p.msgs[t]
	if !ok {
		return 0
	}
	return len(msgs)
}

func (p *P) saveMsg(t string, msg *pubsub.Message) {
	p.msgsLock.Lock()
	defer p.msgsLock.Unlock()

	msgs, ok := p.msgs[t]
	if !ok {
		msgs = make([]*pubsub.Message, 0)
	}
	msgs = append(msgs, msg)
	p.msgs[t] = msgs
}

func newPeers(ctx context.Context, t *testing.T, n int) []*P {
	peers := make([]*P, n)
	for i := 0; i < n; i++ {
		peers[i] = newPeer(ctx, t, 256)
	}
	t.Logf("%d peers were created", n)
	for ctx.Err() == nil {
		done := 0
		for _, p := range peers {
			if atomic.LoadUint64(&p.connsCount) > uint64(n/2) {
				done++
			}
		}
		if done == len(peers) {
			break
		}
	}
	t.Log("peers are connected")
	return peers
}

func newPeer(ctx context.Context, t *testing.T, qSize int) *P {
	host, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	require.NoError(t, err)
	require.NoError(t, discovery.SetupMdnsDiscovery(ctx, zap.L(), host))

	gsParams := pubsub.DefaultGossipSubParams()
	psOpts := []pubsub.Option{
		//pubsub.WithMessageIdFn(n.msgId),
		//pubsub.WithSubscriptionFilter(s),
		pubsub.WithPeerOutboundQueueSize(qSize),
		pubsub.WithValidateQueueSize(qSize),
		pubsub.WithFloodPublish(true),
		pubsub.WithGossipSubParams(gsParams),
		pubsub.WithEventTracer(NewTracer(zap.L(), true)),
	}
	ps, err := pubsub.NewGossipSub(ctx, host, psOpts...)
	require.NoError(t, err)
	tm := NewTopicManager(ctx, zaptest.NewLogger(t), ps, nil)

	p := &P{
		host:     host,
		ps:       ps,
		tm:       tm.(*topicManager),
		msgs:     make(map[string][]*pubsub.Message),
		msgsLock: &sync.Mutex{},
	}
	host.Network().Notify(&libp2pnetwork.NotifyBundle{
		ConnectedF: func(network libp2pnetwork.Network, conn libp2pnetwork.Conn) {
			atomic.AddUint64(&p.connsCount, 1)
		},
	})
	return p
}
