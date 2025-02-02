package p2p

import (
	"context"
	"crypto/rand"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"testing"
)

func TestP2pNetwork_isRelevantPeer(t *testing.T) {
	ctx := context.Background()

	host, pi := newHostWithPeersIndex(ctx, t, testUA)

	relevant := make(map[string]bool)
	lookupHandler := func(oid string) bool {
		return relevant[oid]
	}

	n := &p2pNetwork{
		ctx:            ctx,
		logger:         zaptest.NewLogger(t),
		host:           host,
		peersIndex:     pi,
		lookupOperator: lookupHandler,
		cfg:            &Config{NetworkTrace: true},
	}

	t.Run("identify irrelevant operator", func(t *testing.T) {
		node, info := createPeer(t, Operator)
		pi.IndexNode(node)
		relevant, oid := n.isRelevantPeer(info.ID)
		require.False(t, relevant)
		require.NotEmpty(t, oid)
	})

	t.Run("identify relevant operator", func(t *testing.T) {
		node, info := createPeer(t, Operator)
		pi.IndexNode(node)
		oid, err := extractOperatorIDEntry(node.Record())
		require.NoError(t, err)
		relevant[string(*oid)] = true
		relevant, opid := n.isRelevantPeer(info.ID)
		require.True(t, relevant)
		require.Equal(t, opid, string(*oid))
	})

	t.Run("identify exporter peer", func(t *testing.T) {
		node, info := createPeer(t, Exporter)
		pi.IndexNode(node)
		relevant, oid := n.isRelevantPeer(info.ID)
		require.True(t, relevant)
		require.Empty(t, oid)
	})

	t.Run("handle non-found peer", func(t *testing.T) {
		_, info := createPeer(t, Operator)
		relevant, oid := n.isRelevantPeer(info.ID)
		// currently, accepting unknown peers, this should be changed in the future
		require.True(t, relevant)
		require.Empty(t, oid)
	})
}

func createPeer(t *testing.T, nodeType NodeType) (*enode.Node, *peer.AddrInfo) {
	priv, _, err := crypto.GenerateSecp256k1Key(rand.Reader)
	require.NoError(t, err)
	pk := convertFromInterfacePrivKey(priv)
	ip, err := ipAddr()
	pubkey := genPublicKey()
	require.NoError(t, err)
	node, err := createLocalNode(pk, ip, 12000, 13000)
	require.NoError(t, err)

	node, err = addNodeTypeEntry(node, nodeType)
	require.NoError(t, err)
	if nodeType == Operator {
		node, err = addOperatorIDEntry(node, operatorID(pubkey.SerializeToHexStr()))
		require.NoError(t, err)
	}

	info, err := convertToAddrInfo(node.Node())
	require.NoError(t, err)

	return node.Node(), info
}
