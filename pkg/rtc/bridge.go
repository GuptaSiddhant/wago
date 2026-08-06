// Package rtc is a Media Bridge for VoIP calls in the inbox.
//
// It runs the *remote* endpoint of a WebRTC session so a React client can place
// and answer calls directly in the browser. The client always acts as the
// offerer; the bridge answers with an SDP answer. For the MVP the bridge's own
// audio side is generated in-process (a gentle tone) so media provably flows
// end-to-end without needing a PSTN trunk; swapping that tone for a real
// call-leg source (e.g. a SIP/Twilio bridge) only changes the media source.
package rtc

import (
	"github.com/pion/webrtc/v4"
)

// defaultICEServers make the bridge reachable across NAT while avoiding the
// need for a TURN server on same-LAN / typical deployments.
var defaultICEServers = []webrtc.ICEServer{
	{URLs: []string{"stun:stun.l.google.com:19302"}},
}

// Bridge manages one call's WebRTC peer connection (the remote side).
type Bridge struct {
	CallID string
	pc     *webrtc.PeerConnection
	track  *webrtc.TrackLocalStaticSample
	tone   *toneLoop
}

// newPeerConnection builds a P2P audio-only peer connection pre-configured with
// a STUN-only ICE config.
func newPeerConnection() (*webrtc.PeerConnection, error) {
	var engine webrtc.MediaEngine
	if err := engine.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&engine))

	config := webrtc.Configuration{ICEServers: defaultICEServers}
	return api.NewPeerConnection(config)
}

// NewBridge creates a bridge for a call and starts sourcing a remote audio tone
// so the agent hears something on answer. Close tears the call down.
func NewBridge(callID string) (*Bridge, error) {
	pc, err := newPeerConnection()
	if err != nil {
		return nil, err
	}

	track, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio", "pion",
	)
	if err != nil {
		_ = pc.Close()
		return nil, err
	}
	if _, err := pc.AddTrack(track); err != nil {
		_ = pc.Close()
		return nil, err
	}

	b := &Bridge{CallID: callID, pc: pc, track: track}

	// Consume the agent's (inbound) audio so the receiving pipeline isn't
	// starved — a future real call-leg would forward these frames.
	pc.OnTrack(func(rt *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		for {
			if _, _, err := rt.ReadRTP(); err != nil {
				return
			}
		}
	})

	return b, nil
}

// Answer applies a browser offer and returns the caller's SDP answer. It blocks
// until ICE gathering completes so the answer includes all remote candidates
// (no trickle handshake needed from the client).
func (b *Bridge) Answer(offerSDP string) (string, error) {
	offer := webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: offerSDP}
	if err := b.pc.SetRemoteDescription(offer); err != nil {
		return "", err
	}
	answer, err := b.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	done := webrtc.GatheringCompletePromise(b.pc)
	if err := b.pc.SetLocalDescription(answer); err != nil {
		return "", err
	}
	<-done
	if b.tone == nil {
		b.tone = startToneLoop(b.track)
	}
	return b.pc.LocalDescription().SDP, nil
}

// Close tears the call down and stops the tone.
func (b *Bridge) Close() error {
	if b.tone != nil {
		b.tone.stop()
		b.tone = nil
	}
	return b.pc.Close()
}
