package main

import (
	"encoding/json"
	"fmt"
	"github.com/pion/webrtc/v3"
	"io"
	"log"
	"net"
	"os"
	"pion-to-pion/pkg"
	"pion-to-pion/trans"
	"sync"
)

func main() {
	listener, err := net.Listen("tcp", ":2222")
	if err != nil {
		panic(err)
	}
	fmt.Println("Listener listening on port 2222")
	for {
		l, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func() {
			RunWebRTCAnswer(l)
		}()
	}
}

func RunWebRTCAnswer(sock net.Conn) {
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	// Everything below is the Pion WebRTC API! Thanks for using it ❤️.
	t := trans.NewTrans("ws://66.42.61.19:8090/ws")
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
		if err := peerConnection.Close(); err != nil {
			fmt.Printf("cannot close peerConnection: %v\n", err)
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
				fmt.Println("Sdp........................", m.SDP)
				sdp := webrtc.SessionDescription{}
				if err := json.Unmarshal([]byte(m.SDP), &sdp); err != nil {
					panic(err)
				}

				if err := peerConnection.SetRemoteDescription(sdp); err != nil {
					panic(err)
				}

				// Create an answer to send to the other process
				answer, err := peerConnection.CreateAnswer(nil)
				if err != nil {
					panic(err)
				}

				// Send our answer to the HTTP server listening in the other process
				payload, err := json.Marshal(answer)
				if err != nil {
					panic(err)
				}
				var sdpMess = trans.Message{
					Type: "sdp",
					SDP:  string(payload),
				}

				t.GetSend() <- sdpMess
				// resp, err := http.Post(fmt.Sprintf("http://%s/sdp", *offerAddr), "application/json; charset=utf-8", bytes.NewReader(payload)) // nolint:noctx
				// if err != nil {
				// 	panic(err)
				// } else if closeErr := resp.Body.Close(); closeErr != nil {
				// 	panic(closeErr)
				// }

				// Sets the LocalDescription, and starts our UDP listeners
				err = peerConnection.SetLocalDescription(answer)
				if err != nil {
					panic(err)
				}

				candidatesMux.Lock()
				for _, c := range pendingCandidates {
					t.GetSend() <- trans.Message{
						Type:      "cad",
						Candidate: c.ToJSON().Candidate,
					}
				}
				candidatesMux.Unlock()
			case "cad":
				fmt.Println(m.Candidate)
				if candidateErr := peerConnection.AddICECandidate(webrtc.ICECandidateInit{Candidate: m.Candidate}); candidateErr != nil {
					panic(candidateErr)
				}
			}
		}
	}()
	peerConnection.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		fmt.Printf("Peer Connection State has changed: %s\n", s.String())

		if s == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			fmt.Println("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}
	})
	// Register data channel creation handling
	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		fmt.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		// Register channel opening handling
		d.OnOpen(func() {
			fmt.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
			io.Copy(pkg.NewDataChanelWriter(d), sock)
			d.Close()
			log.Println("disconnected")
		})

		// Register text message handling
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			_, err := sock.Write(msg.Data)
			if err != nil {
				fmt.Println("error writing message", err)
				log.Println("disconnected")
			}
		})
	})

	select {}
}
