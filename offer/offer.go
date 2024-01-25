package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"io"
	"net"
	"pion-to-pion/pkg"
	"pion-to-pion/trans"
	"sync"
)

func main() {
	RunWebRTCOffer()
}

func RunWebRTCOffer() {
RESET:
	var ResetSignal = struct{}{}
	var reset = make(chan struct{})
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	var ssh net.Conn
	t := trans.NewTrans("ws://66.42.61.19:8090/ws")
	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.

	// Prepare the configuration
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	// Create a new RTCPeerConnection
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if cErr := peerConnection.Close(); cErr != nil {
			fmt.Printf("cannot close peerConnection: %v\n", cErr)
		}
	}()

	peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}

		candidatesMux.Lock()
		defer candidatesMux.Unlock()

		desc := peerConnection.RemoteDescription()
		if desc == nil {
			pendingCandidates = append(pendingCandidates, c)
		} else {
			t.GetSend() <- trans.Message{
				Type:      "cad",
				Candidate: c.ToJSON().Candidate,
			}
		}
	})

	go func() {
		for m := range t.GetMess() {
			switch m.Type {
			case "sdp":
				sdp := webrtc.SessionDescription{}
				if sdpErr := json.Unmarshal([]byte(m.SDP), &sdp); sdpErr != nil {
					panic(sdpErr)
				}
				fmt.Println(sdp)

				if sdpErr := peerConnection.SetRemoteDescription(sdp); sdpErr != nil {
					panic(sdpErr)
				}

				candidatesMux.Lock()
				defer candidatesMux.Unlock()

				for _, c := range pendingCandidates {
					t.GetSend() <- trans.Message{
						Type:      "cad",
						Candidate: c.ToJSON().Candidate,
					}
				}
			case "cad":
				fmt.Println(m.Candidate)
				if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: m.Candidate}); candidateErr != nil {
					panic(candidateErr)
				}
			}
		}
	}()

	// Create a datachannel with label 'data'
	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			reset <- ResetSignal
		}
	})

	// Register channel opening handling
	dataChannel.OnOpen(func() {
		fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", dataChannel.Label(), dataChannel.ID())
		ssh, err = net.Dial("tcp", "localhost:22")
		if err != nil {
			fmt.Println("Dial error, Returning")
			reset <- ResetSignal
			return
		}
		io.Copy(pkg.NewDataChanelWriter(dataChannel), ssh)
		fmt.Println("Disconnected")
		reset <- ResetSignal
	})
	// Register text message handling
	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if ssh != nil {
			_, err := ssh.Write(msg.Data)
			if err != nil {
				fmt.Println("Disconnected")
				reset <- ResetSignal
			}

		}
	})

	// Create an offer to send to the other process
	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}

	if err = peerConnection.SetLocalDescription(offer); err != nil {
		panic(err)
	}

	// Send our offer to the HTTP server listening in the other process
	payload, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}

	//Send sdp
	var sdpMess = trans.Message{
		Type: "sdp",
		SDP:  string(payload),
	}

	t.GetSend() <- sdpMess

	// Block forever
	select {
	case <-reset:
		goto RESET
	}
}
