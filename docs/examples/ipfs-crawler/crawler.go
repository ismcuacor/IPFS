package main

import (
	"context"
	"fmt"
	"io/ioutil"
	//"io"
	"log"
        "path/filepath"

	"container/list"

	config "github.com/ipfs/go-ipfs-config"
	libp2p "github.com/ipfs/go-ipfs/core/node/libp2p"
	icore "github.com/ipfs/interface-go-ipfs-core"

	core "github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
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

func main() {
	//checkSwarmAPI()
	checkSwarmHTTP()

	for (true) {
		for peer := peersList.Front(); peer != nil; peer = peer.Next() {
			if _,hit := peersMap[peer.Value.(string)]; !hit {
				//findClosestPeersAPI(peer.Value.(string))
				findClosestPeersHTTP(peer.Value.(string))
			}
		}
	//	time.Sleep(10 * time.Second) // uncomment if we want to give a break to the system
	}
}

func checkSwarmAPI (){
	ctx = context.Background()

	ipfs, err := spawn(ctx)
	logError(err, "retrieving swarm")

	checkSwarmHTTP()

	peersSwarmAPI, err := ipfs.Swarm().Peers(ctx)
	logError(err, "retrieving swarm")

	for _,peer := range peersSwarmAPI {
		peersList.PushBack(peer.ID().Pretty())
	}

	fmt.Println("Nodes in the swarm", peersList.Len())
}

func findClosestPeersAPI(peer string, ctx context.Context, ipfs icore.CoreAPI) {
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

                peersMap[nextPeer.Pretty()] = 1
        }

        fmt.Println("Nodes in the map", len(peersMap))
        fmt.Println("Churn until now ", churn/len(peersMap))
}

