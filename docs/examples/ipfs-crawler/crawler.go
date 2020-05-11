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

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/plugin/loader"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
)

// All peers which have been discovered so far
var peersList = list.New()
var peersMap = make(map[string]int)

// To beautify errors and help debbugging & reading
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

	node, err := core.NewNode(ctx, nodeOptions)
	logError(err, "creating new node")

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
	logError(err, "creating temp repo")

	// Spawning an IPFS node
	return createNode(ctx, repoPath)
}

func main() {
	//checkSwarmAPI(ctx, ipfs)

	for (true) {
		for peer := peersList.Front(); peer != nil; peer = peer.Next() {
			if _,hit := peersMap[peer.Value.(string)]; !hit {
				findClosestPeers(peer.Value.(string))
			}
		}
	//	time.Sleep(10 * time.Second) // uncomment if we want to give a break to the system
	}
}

func checkSwarmAPI (ctx context.Context, ipfs icore.CoreAPI){
	fmt.Println("-- Getting an IPFS node running -- ")

	ctx = context.Background()

	fmt.Println("Spawning node")
	ipfs, err := spawn(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to spawn node: %s", err))
	}

	fmt.Println("IPFS node is running")

	checkSwarmHTTP()

	peersSwarmAPI, err := ipfs.Swarm().Peers(ctx)
	logError(err, "retrieving swarm")

	for _,peer := range peersSwarmAPI {
		peersList.PushBack(peer.ID().Pretty())
	}

	fmt.Println("Nodes in the swarm", peersList.Len())
}

