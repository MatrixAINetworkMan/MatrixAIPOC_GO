## Introduction

This is considered as a POC of Matrix in the form of golang implementation based on the Ethereum protocol.

### In this POC version (v0.0.1), MATRIX made such improvements:

##### Consensus: node election and verification node design

1. Fully support the generation of Computing Power Output Network 

2. Fully support the generation of Verification Node Network as well as the updates of network status

##### WHAT'S NEW ON P2P

1. Periodic broadcasts of selected nodes: selection results will be broadcast to the peer nodes

2. Full nodes will insert the node selection information into globallist upon receiving and verifying the selection results

3. Periodic refresh of globallist and removal of dropped nodes or nodes facing unsteady network

4. Package the globallist into block and broadcast both the block and the globallist following successful mining; local globallist will be updated after the block info is synchronized 


### In this POC version (v0.0.2), MATRIX introduced:


- a new boot process
- a new static sychronization process
- a new election algorithm
- a new scheduler module as well as network topology generation process
- a new validator node module
- a masternode list for block head
- a new process for election transaction and quitting from election



