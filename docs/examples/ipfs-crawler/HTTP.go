package main

import (
	"fmt"
	"io/ioutil"
	"crypto/tls"
	"strings"
	"net/http"
	"encoding/json"
	"os"
	"io"
	"bufio"
)

//JSON response for REST query
type PeersSwarm struct {
	Peers []struct {
		Addr      string `json:"Addr"`
		Peer      string `json:"Peer"`
		Latency   string `json:"Latency"`
		Muxer     string `json:"Muxer"`
		Direction int    `json:"Direction"`
		Streams   []struct {
			Protocol string `json:"Protocol"`
		} `json:"Streams"`
	} `json:"Peers"`
}

type Query struct {
	Extra     string      `json:"Extra"`
	ID        string      `json:"ID"`
	Responses interface{} `json:"Responses"`
	Type      int         `json:"Type"`
}

func checkSwarmHTTP () {
	uri := "http://127.0.0.1:5001/api/v0/swarm/peers?verbose=true&streams=true&latency=true&direction=true"
	requestString := strings.NewReader("")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", uri, requestString)
	logError(err, "requesting swarm")

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)

	logError(err, "receiving swarm")

	var peersSwarm PeersSwarm
	responseBytes, err := ioutil.ReadAll(resp.Body)

	logError(err, "reading swarm")

	err = json.Unmarshal(responseBytes, &peersSwarm)

	logError(err, "unmarshalling swarm")

	for _,peer := range peersSwarm.Peers {
		peersList.PushFront(peer.Peer)
	}

	defer resp.Body.Close()

	fmt.Println("Nodes in the swarm", peersList.Len())
}

func findClosestPeersHTTP(peerID string) {
	uri := "http://127.0.0.1:5001/api/v0/dht/query?arg="+peerID
	file := "filename.txt"

	requestString := strings.NewReader("")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", uri, requestString)
	logError(err, "requesting peers for peer "+ peerID)

	resp, err := client.Do(req)
	logError(err, "geeting peers for peer "+ peerID)

	out, err := os.Create(file)
	logError(err, "create temp file "+ peerID)
	defer out.Close()

	io.Copy(out, resp.Body)
	logError(err, "copying Body")

	defer resp.Body.Close()
	logError(err, "closing response")

	closePeers := make(map[int]Query)
	parseFileToResponse(file, closePeers)
	logError(err, "erasing temp file")

	err = os.Remove(file)

	for _, nextPeer := range closePeers {
		//When there's Extra, is to say that no address was reachable
		if _, hit := peersMap[nextPeer.ID]; nextPeer.Extra != "" && !hit  {
			// Add the peer to the list of nodes to visit in the next Iteration
			peersList.PushBack(nextPeer)
			
			churn++
		}
		
		peersMap[nextPeer.ID] = 1
	}

	fmt.Println("Nodes in the map", len(peersMap))
        fmt.Println("Down nodes until now ", churn)
	fmt.Println("Churn until now ", churn/len(peersMap))
}

func parseFileToResponse(file string, closePeers map[int]Query) {
	readFile, err := os.Open(file)
	logError(err, "opening file")

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileTextLines []string

	for fileScanner.Scan() {
		fileTextLines = append(fileTextLines, fileScanner.Text())
	}

	readFile.Close()

	i:=0
	var nextPeer Query
	for _, eachline := range fileTextLines {
		err = json.Unmarshal([]byte(eachline), &nextPeer)
		logError(err, "unmarshalling from file")

		closePeers[i] = nextPeer
		i++
	}
}
