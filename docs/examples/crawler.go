package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"crypto/tls"
	"path/filepath"
	"strings"
	"net/http"
	"encoding/json"
	"os"
	//"io"
	"log"
	"bufio"

	config "github.com/ipfs/go-ipfs-config"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
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

var churn = 0

// All peers which have been discovered so far
var peersAPI map[string]int
var peersList map[string]int

/// ------ Setting up the IPFS Repo
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err := plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err := plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func CreateRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
	if err != nil {
		return "", fmt.Errorf("failed to get temp dir: %s", err)
	}

	// Create a config with default options and a 2048 bit key
	cfg, err := config.Init(ioutil.Discard, 2048)
	if err != nil {
		return "", err
	}

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to init ephemeral node: %s", err)
	}

	return repoPath, nil
}

/// ------ Spawning the node

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTClientOption, // This option sets the node to be a client DHT node (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(node)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawn(ctx context.Context) (icore.CoreAPI, error) {
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Repo
	repoPath, err := CreateRepo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	}

	// Spawning an IPFS node
	return createNode(ctx, repoPath)
}

func main() {
	peersAPI = make(map[string]int)
	peersList = make(map[string]int)

	//fmt.Println("-- Getting an IPFS node running -- ")

	//ctx := context.Background()

	//fmt.Println("Spawning node")
	//ipfs, err := spawn(ctx)
	//if err != nil {
	//	panic(fmt.Errorf("failed to spawn node: %s", err))
	//}

	//fmt.Println("IPFS node is running")

	fmt.Println("Checking over HTTP")
	checkSwarmHTTP()

	//fmt.Println("Checking over API")
	//checkSwarmAPI(ctx, ipfs)

	//for (true) {
		for peer, state := range peersList {
			if state < 2 {
				findClosestPeers(peer)
				peersList[peer] = 2
			}
		}
	//	time.Sleep(10 * time.Second)
	//}
}

func checkSwarmAPI (ctx context.Context, ipfs icore.CoreAPI){
	peersSwarmAPI, err := ipfs.Swarm().Peers(ctx)
	if err != nil {
	    fmt.Println(err.Error())
	}

	newPeers := 0
	for _,peer := range peersSwarmAPI {
		_, hit := peersAPI[peer.ID().Pretty()]
		if !hit {
			newPeers ++
		}
		peersAPI[peer.ID().Pretty()] = 1
	}

	fmt.Println("Nodes in the swarm", len(peersAPI))
	fmt.Println("New peers", newPeers)
}

func checkSwarmHTTP () {
	uri := "http://127.0.0.1:5001/api/v0/swarm/peers?verbose=true&streams=true&latency=true&direction=true"
	requestString := strings.NewReader("")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", uri, requestString)
	if err != nil {
	    fmt.Println(err.Error())
	}

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)

	if err != nil {
	    fmt.Println(err.Error())
	}

	var peersSwarm PeersSwarm
	responseBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err.Error())
	}

	err = json.Unmarshal(responseBytes, &peersSwarm)

	if err != nil {
		fmt.Println(err.Error())
	}

	newPeers := 0
	for _,peer := range peersSwarm.Peers {
		_, hit := peersList[peer.Peer]
		if !hit {
			newPeers ++
		}
		peersList[peer.Peer] = 1
	}

	defer resp.Body.Close()


	fmt.Println("Nodes in the swarm", len(peersList))
	fmt.Println("New peers", newPeers)
}

func findClosestPeers(peerID string) {
	uri := "http://127.0.0.1:5001/api/v0/dht/query?arg="+peerID
	file := "filename.txt"

	requestString := strings.NewReader("")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("POST", uri, requestString)
	if err != nil {
	    fmt.Println(err.Error())
	}

	resp, err := client.Do(req)

	if err != nil {
	    fmt.Println(err.Error())
	}

	out, err := os.Create(file)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer out.Close()
	io.Copy(out, resp.Body)

	defer resp.Body.Close()

	if err != nil {
		fmt.Println(err.Error())
	}

	closePeers := make(map[int]Query)
	parseFileToResponse(file, closePeers)

	err := os.Remove(file)

	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}

	newNodes := 0
	//When there's Extra, is to say that no address was reachable
	for _, nextPeer := range closePeers {
		if nextPeer.Extra != "" && peersList[nextPeer.ID] < 1 {
			churn++
		}

		_, hit := peersList[nextPeer.ID]
		if !hit {
			newNodes++
			peersList[nextPeer.ID] = 1
		}
	}

	fmt.Println("Nodes in the list", len(peersList))
	fmt.Println("New Nodes", newNodes)
	fmt.Println("Churn until now ", churn)
}

func parseFileToResponse(file string, closePeers map[int]Query) {
	readFile, err := os.Open(file)

	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}

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
		if err != nil {
			log.Fatalf("failed to open file: %s", err)
		}
		closePeers[i] = nextPeer
		i++
	}
}
