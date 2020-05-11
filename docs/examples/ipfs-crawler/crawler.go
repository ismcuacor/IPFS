package main

import (
	"context"
	"fmt"
	"io/ioutil"
	//"io"
	"log"
        "path/filepath"
	"sync"
	"container/list"

	config "github.com/ipfs/go-ipfs-config"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"
	core "github.com/ipfs/go-ipfs/core"
	peerstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"github.com/libp2p/go-libp2p-core/peer"
)

// All peers which have been discovered so far
var peersList = list.New()
var peersMap = make(map[string]int)
var myNode *core.IpfsNode
var ctx context.Context
var ipfs icore.CoreAPI

// To prettify errors and help debbugging & reading
func logError(err error, str string) {
	if err != nil {
		log.Printf("Failed at %s with error %s", str, err)
	}
	err = nil
}

/// ------ Setting up the IPFS Repo
func setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
        logError(err, "loading plugins")

	// Load preloaded and external plugins
	err = plugins.Initialize()
        logError(err, "initializing plugins")

	err = plugins.Inject()
        logError(err, "injecting plugins")

	return nil
}

func CreateRepo(ctx context.Context) (string, error) {
	repoPath, err := ioutil.TempDir("", "ipfs-shell")
        logError(err, "opening temp dir")

	cfg, err := config.Init(ioutil.Discard, 2048)
        logError(err, "creating a config with default options and a 2048 bit key")

	// Create the repo with the config
	err = fsrepo.Init(repoPath, cfg)
	logError(err, "creating the repo for node")

	return repoPath, nil
}

/// ------ Spawning the node

// Creates an IPFS node and returns its coreAPI
func createNode(ctx context.Context, repoPath string) (icore.CoreAPI, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	logError(err, "opening the repo")

	// Construct the node
	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo: repo,
	}

	myNode, err = core.NewNode(ctx, nodeOptions)
	logError(err, "creating new node")

	// Attach the Core API to the constructed node
	return coreapi.NewCoreAPI(myNode)
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func spawn(ctx context.Context) (icore.CoreAPI, error) {
	if err := setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Repo
	repoPath, err := CreateRepo(ctx)
	logError(err, "creating temp repo")

	// Spawning an IPFS node
	return createNode(ctx, repoPath)
}

//

func connectToPeers(peers []string) error {
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peerstore.PeerInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peerstore.PeerInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	wg.Add(len(peerInfos))
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peerstore.PeerInfo) {
			defer wg.Done()
			err := ipfs.Swarm().Connect(ctx, *peerInfo)
			if err != nil {
				log.Printf("failed to connect to %s: %s", peerInfo.ID, err)
			}
		}(peerInfo)
	}
	wg.Wait()
	return nil
}

func main() {
	startIPFS()

	bootstrapNodes := []string{
		// IPFS Bootstrapper nodes.
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
		"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",

		// IPFS Cluster Pinning nodes
		"/ip4/138.201.67.219/tcp/4001/p2p/QmUd6zHcbkbcs7SMxwLs48qZVX3vpcM8errYS7xEczwRMA",
		"/ip4/138.201.67.220/tcp/4001/p2p/QmNSYxZAiJHeLdkBg38roksAR9So7Y5eojks1yjEcUtZ7i",
		"/ip4/138.201.68.74/tcp/4001/p2p/QmdnXwLrC8p1ueiq2Qya8joNvk3TVVDAut7PrikmZwubtR",
		"/ip4/94.130.135.167/tcp/4001/p2p/QmUEMvxS2e7iDrereVYc5SWPauXPyNwxcy9BXZrC1QTcHE",

		// You can add more nodes here, for example, another IPFS node you might have running locally, mine was:
		// "/ip4/127.0.0.1/tcp/4010/p2p/QmZp2fhDLxjYue2RiUvLwT9MWdnbDxam32qYFnGmxZDh5L",
	}
	//To make sure that the swarm is well connected

	go connectToPeers(bootstrapNodes)

	checkSwarmAPI()
	//checkSwarmHTTP()

	for (true) {
		fmt.Println("Iteration ")
		for peer := peersList.Front(); peer != nil; peer = peer.Next() {
			if _,hit := peersMap[peer.Value.(string)]; !hit {
				findClosestPeersAPI(peer.Value.(string))
				//findClosestPeersHTTP(peer.Value.(string))
			}
		}
	//	time.Sleep(10 * time.Second) // uncomment if we want to give a break to the system
	}
}

func startIPFS() {
	var err error
	ctx = context.Background()

	ipfs, err = spawn(ctx)
	logError(err, "retrieving swarm")
}

func checkSwarmAPI (){
	peersSwarmAPI, err := ipfs.Swarm().Peers(ctx)
	logError(err, "retrieving swarm")

	for _,peer := range peersSwarmAPI {
		peersList.PushBack(peer.ID().Pretty())
	}

	fmt.Println("Nodes in the swarm", peersList.Len())
}

func findClosestPeersAPI(peer string) {
	dht := myNode.DHT.WAN
	peers, err := dht.GetClosestPeers(ctx, peer)
	logError(err, "retrieving closest peers")

        for nextPeer := range peers {
		peerInfo, err := dht.FindPeer(ctx, nextPeer)
		logError(err, "retrieving swarm")
                //To check for churn, we need to try and connect to the peer in any address
		err = ipfs.Swarm().Connect(ctx, peerInfo)

		if err != nil {
                        churn++
                }
		//TODO create here a function to delete unnecesary connections, looking at connectInfo

                peersMap[nextPeer.Pretty()] = 1
        }

        fmt.Println("Nodes in the map", len(peersMap))
        fmt.Println("Churn until now ", churn/len(peersMap))
}

