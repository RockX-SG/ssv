package p2p

import (
	"github.com/libp2p/go-libp2p-core/control"
	libp2pnetwork "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// Implementing ConnectionGater interface.
// https://github.com/libp2p/go-libp2p-core/blob/master/connmgr/gater.go

// InterceptPeerDial is called on an imminent outbound peer dial request, prior
// to the addresses of that peer being available/resolved. Blocking connections
// at this stage is typical for blacklisting scenarios
func (n *p2pNetwork) InterceptPeerDial(id peer.ID) bool {
	return true
}

// InterceptAddrDial is called on an imminent outbound dial to a peer on a
// particular address. Blocking connections at this stage is typical for
// address filtering.
func (n *p2pNetwork) InterceptAddrDial(id peer.ID, multiaddr ma.Multiaddr) bool {
	return true
}

// InterceptAccept is called as soon as a transport listener receives an
// inbound connection request, before any upgrade takes place. Transports who
// accept already secure and/or multiplexed connections (e.g. possibly QUIC)
// MUST call this method regardless, for correctness/consistency.
func (n *p2pNetwork) InterceptAccept(multiaddrs libp2pnetwork.ConnMultiaddrs) bool {
	return true
}

// InterceptSecured is called for both inbound and outbound connections,
// after a security handshake has taken place and we've authenticated the peer.
//
func (n *p2pNetwork) InterceptSecured(direction libp2pnetwork.Direction, id peer.ID, multiaddrs libp2pnetwork.ConnMultiaddrs) bool {
	if pruned := n.peersIndex.Pruned(id); pruned {
		n.trace("rejecting pruned peer", zap.String("who", "conn_gater"),
			zap.String("pid", id.String()))
		return false
	}
	return true
}

// InterceptUpgraded is called for inbound and outbound connections, after
// libp2p has finished upgrading the connection entirely to a secure,
// multiplexed channel.
func (n *p2pNetwork) InterceptUpgraded(conn libp2pnetwork.Conn) (bool, control.DisconnectReason) {
	return true, 0
}
