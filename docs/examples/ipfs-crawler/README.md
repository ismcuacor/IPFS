# IPFS
Example of a IPFS crawler. 

This go file shows how to build a crawler for IPFS, which parses the swarm of nodes in the network. This code uses both the HTTP REST and API to evaluate the performance of both. It evaluates three metrics: Number of peers, new peers (since the last check up) and churn

### Number of peers
It represents how many peers have been found in the swarm

### Churn
It represents the percentage of peers which have left the network since the last checkup
