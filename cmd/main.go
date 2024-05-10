package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"log"

	"github.com/pion/webrtc/v4"
)

const signalingServer = "http://localhost:8080"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go [send|receive]")
		os.Exit(1)
	}

	mode := os.Args[1]
	switch mode {
	case "send":
		handleSend()
	case "receive":
		handleReceive()
	default:
		fmt.Println("Invalid mode specified. Use 'send' or 'receive'.")
		os.Exit(1)
	}
}

func handleSend() {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
	if err != nil {
		panic(err)
	}

	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	open := make(chan struct {})
	dataChannel.OnOpen(func ()  {
		fmt.Println("Data channel opened")
		close(open)
	})
	<-open

	fileBytes, err := ioutil.ReadFile("file.txt")
	if err != nil {
		panic(err)
	}

	dataChannel.Send(fileBytes)
	fmt.Println("File Sent")

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	peerConnection.SetLocalDescription(offer)

	sendOffer(offer)

	answer := getAnswer()
	peerConnection.SetRemoteDescription(answer)

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
	<-gatherComplete

	select {}
}

func handleReceive() {
	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{})
    if err != nil {
        panic(err)
    }

	peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
        d.OnMessage(func(msg webrtc.DataChannelMessage) {
            fmt.Println("Received message:", string(msg.Data))
            ioutil.WriteFile("received_file.txt", msg.Data, 0644)
            fmt.Println("File written to received_file.txt")
        })
    })

	offer := getOffer()
    peerConnection.SetRemoteDescription(offer)

	answer, err := peerConnection.CreateAnswer(nil)
    if err != nil {
        panic(err)
    }
    peerConnection.SetLocalDescription(answer)

	sendAnswer(answer)

	gatherComplete := webrtc.GatheringCompletePromise(peerConnection)
    <-gatherComplete

    select {}
}

func sendOffer(offer webrtc.SessionDescription) {
	body, err := json.Marshal(offer)
	if err != nil {
		panic(err)
	}
	_, err = http.Post(signalingServer+"/offer", "application/json", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
}

func getAnswer() webrtc.SessionDescription {
	resp, err := http.Get(signalingServer+"/answer")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var answer webrtc.SessionDescription
	if err := json.NewDecoder(resp.Body).Decode(&answer); err != nil {
		panic(err)
	}
	return answer
}

func getOffer() webrtc.SessionDescription { 
	// resp, err := http.Get(signlaingServer+"/offer")
	// if err != nil {
	// 	panic(err)
	// }
	// defer resp.Body.Close()
	// var offer webrtc.SessionDescription
	// if err := json.NewDecoder(resp.Body).Decode(&offer); err != nil {
	// 	panic(err)
	// }
	// return offer

	resp, err := http.Get(signalingServer + "/offer")
    if err != nil {
        log.Fatalf("Error getting offer: %v", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Error reading response body: %v", err)
    }

    fmt.Printf("Received raw response: %s\n", string(body))

    var offer webrtc.SessionDescription
    if err := json.Unmarshal(body, &offer); err != nil {
        log.Fatalf("Error decoding offer: %v", err)
    }
    return offer

}

func sendAnswer(answer webrtc.SessionDescription) {
	body, err := json.Marshal(answer)
	if err != nil {
		panic(err)
	}
	_, err = http.Post(signalingServer+"/answer", "application/json", bytes.NewReader(body))
	if err != nil {
		panic(err)
	}
}