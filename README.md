## Introduction

This is considered as a POC of Matrix in the form of golang implementation based on the Ethereum protocol.

### In this POC version, Matrix made such improvements:

##### Consensus: node election and verification node design

1. Fully support the generation of Computing Power Output Network 

2. Fully support the generation of Verification Node Network as well as the updates of network status

##### WHAT'S NEW ON P2P

1. Periodic broadcasts of selected nodes: selection results will be broadcast to the peer nodes

2. Full nodes will insert the node selection information into globallist upon receiving and verifying the selection results

3. Periodic refresh of globallist and removal of dropped nodes or nodes facing unsteady network

4. Package the globallist into block and broadcast both the block and the globallist following successful mining; local globallist will be updated after the block info is synchronized 


##### PROGRESS UPDATE

2018.6.21:

- Finalized the plan of block generation and transaction validation

- TCP/UDP related modules as well as interfaces in dev

- Miner module in coding 
