# Iris: A Global P2P network for Sharing Threat Intelligence

Iris is a P2P system for collaborative defense proposed by Bc. Martin Řepa as a [diploma thesis work](https://www.stratosphereips.org/thesis-projects-list/2022/3/12/global-permissionless-p2p-system-for-sharing-distributed-threat-intelligence) (see  the thesis for theoretical details).
This repository hosts a reference implementation written in Golang using [LibP2P project](https://github.com/libp2p) along with integration of Iris
into [Slips IPS](https://github.com/draliii/StratosphereLinuxIPS) and [Fides Trust Model](https://github.com/lukasforst/fides). 

For more details regarding architecture/implementation, we refer reader to [docs/architecture.md](docs/architecture.md) or the thesis itself.

### Motivation 

_shortened thesis abstract:_

Despite the severity and amount of daily cyberattacks, the best solutions our community has so far are
centralised, threat intelligence shared lists; or centralised, commercially-based defence products.
No system exists yet to automatically connect endpoints globally and share information about new attacks
to improve their security. 

Iris allows collaborative defence in cyberspace with emphasis on security and privacy concerns.
It is a pure and completely decentralised P2P network that allows peers to (i) share threat intelligence
files, (ii) alert peers about detected attacks, and (iii) ask peers about their opinion on potential
attacks. Iris addresses the problem of confidentiality of local threat intelligence data by
introducing the concept of _Organisations_. Organisations are cryptographically-verified and
trusted groups of peers within the P2P network. They allow Iris to send content only
to pre-trusted groups of peers.

## Dependencies

To run a standalone peer, you need:
* a running redis instance
* golang (>1.17)

## User Guide

### OrgSig Tool

For pleasure manipulation with organisations, we present a tool called **orgsig**. Orgsig is a small program written in Golang
that can generate organisations or sign existing peers ID using already generated organisation.

```bash
> make orgsig 
go build cmd/orgsig.go
>  ./orgsig --help
Running v0.0.1 orgsig

Usage of ./orgsig:
  -load-key-path string
    	Path to a file with organisation private key. If not set, new private-key is generated.
  -peer-id string
    	Public ID of a peer to sign. Flag --sign-peer must be set for this option to be valid.
  -save-key-path string
    	If set, value will be used as a path to save organisation private-key.
  -sign-peer
    	Flag to sign peer ID. Flag peer-id can be used to set peerID, otherwise, cli will ask. The signature will be printed to stdout.
```


### Running a Peer

Starting a peer with reference configuration is as simple as running (assuming a Redis instance is running on local host):

> make run

### Debugging, Running Multiple Peers

To run silmutaniously multiple peers, you can use already prepared docker-compose file with pre-configured 4 peers.
The network of 4 peers can be started with (note that you must have `docker` installed):

```bash
> make network
```

This command starts docker-compose with 4 peers in separate containers and one container with separate Redis instance. 
Every peer connects to a different Redis channel and waits for messages from Fides (Fides mock has not yet been implemented). 
The peers will connect to each other and thus form a small network. 
Configuration files of every peer can be found in [dev/](dev) directory. 
To interact with the peers, you must act as Fides Trust Model and send to the peers manually a message by publishing some 
messages through Redis channels. Example PUBLISH commands can be found in [dev/redisobj.dev](dev/redisobj.dev).


## Todo/Future Work:
* Fides Trust Model Mock for better testing and debugging
* Complete reference integration of Iris, Fides and Slips inside docker-compose
* Signal handling for graceful shutdown
* After a peer connects to the network, search immediately for members of trustworthy organisations. So far only `connector` does it.
* Implement message (bytes?) rate-limiting per individual peers to mitigate flooding attacks (or adaptive gossips?)
* Use more the Reporting Protocol to report misbehaving peers
* Implement purging of keys after some time (configurable?) in peers' message cache
* responseStorage goroutines should not wait for responses from peers that disconnected during the waiting. Otherwise,
when that happens it's gonna unnecessarily wait until the timeout occurs
* storageResponse goroutines should wait only for responses from peers where requests were successfully sent (err was nil)
* implement purging of file metadata after files expire (viz currently not used field `ElapsedAt`)
* Is reference basic manager really trimming peers based on their reliability? Need to be checked
* **Plus Future Work mentioned in the thesis itself** 
