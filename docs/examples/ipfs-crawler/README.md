# IPFS Nodes Crawler
## Objective
The goal of this application is to use the IPFS API to build a crawler in the IPFS network, retrieving the nodes (Peers) which are part of it.
## Structure
The idea of the crawler is to go over the DHT annotating each neighbor to a certain node. This process is repeated in a Breadth Search algorithm to retrieve the list of all nodes in the network. To avoid cycles, a list of visited nodes is kept, using a hashmap to improve the accessing time for it.

![Breadth-First traverse](./Figure.png)

Similarly, a crawler could be implemented by parsing all the nodes associated to a certain content ID. However, for this approach to be representative, the node running the crawler should have access to a significant part of the content hashkeys (or else try to brute force all possible ones), which is not the goal of the application.

## Implementation 
This code uses both the HTTP REST and Core API. The queries made to the APIs are:
 <p>- Connect(), to create a node and connect it to the network</p>
 <p>- Swarm(), to get the swarm of nodes that are connected to the new node</p>
 <p>- Dht(), to retrieve the DHT table. There are a few considerations here:</p>
       <p><t>-- For the DHT table, the HTTP API (also JS core API and CLI, but those are not implemented here) have access to the neightbors (closest peers) to a node. In the CoreAPI there are 2 objects with access to this table: DhtAPI and routing/DHT. The first one, however, does not have a way to find neighbors </p>
       <p><t>-- A similar crawler (with a different implementation) could be obtained by using the DhtAPI.findProviders() method in the go-Core API, which returns the peers hosting a specific file.</p> 
       <p><t>-- The dht/get method used from the HTTP API returns a set of JSONs (one per neightbor). This message is too big for hosting it in the memory (specially for resource limited computers), so it is backed-up in a temporal file for processing. </p>

Also, it is a interesting work seeing how the HTTP and CoreAPI behave when working on the same Swarm

## Running the application 
For the app to function, the ipfs deamon shall be running with "ipfs daemon &". Then, the application is run with "go run . ". It shows two metrics, each time that it processes a Node: lenght of the Hash Table so far (discovered nodes) and churn (which of these nodes are disconnected from the network.

## Discussion
It is obviously not the goal of IPFS to have a tracker of nodes in the system, but it is still possible (given enough resources) to build this list by traversing the tree of nodes. More information (not only IDs is equally accesible to the programmer (such as the addresses pointing to a node, and can be retrieved. There's also an ongoing discussion inside the community to abandon the DHTAPI object in favour of routing ones, due to the low performance of this object.

Finally, the perfomance of the algorithm is linear in space and time. This performance (in time) is the best that can be achieved, since all nodes need to be visited at least one time.

## More information
 <p>- IPFS HTTP API: https://docs.ipfs.io/reference/api/http/</p>
 <p>- IPFS JSCoreAPI: https://docs.ipfs.io/reference/api/libraries/</p>
 <p>- IPFS CLI: https://docs.ipfs.io/reference/api/cli</p>
 <p>- GO DHTAPI: https://github.com/ipfs/go-ipfs/blob/master/core/coreapi/dht.go</p>
 <p>- GO routing: https://godoc.org/github.com/multikatt/go-ipfs/routing/dht</p>
