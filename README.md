# Iris: A Global P2P network for Sharing Threat Intelligence

Iris is a P2P system for collaborative defense proposed by Bc. Martin Řepa as a [diploma thesis work](https://www.stratosphereips.org/thesis-projects-list/2022/3/12/global-permissionless-p2p-system-for-sharing-distributed-threat-intelligence).
This repository hosts a reference implementation written in Golang using [LibP2P project](https://github.com/libp2p) along with integration of Iris
into [Stratosphere Linux IPS](https://github.com/draliii/StratosphereLinuxIPS) with [Fides Trust Model](https://github.com/lukasforst/fides). 
See a description of all components in the environment and general architecture overview in [See the Architecture Overview](architecture.md).

## Features

### Core Protocols 
Iris offers 3 major protocols exchanging threat intelligence to support collaborative defense:

* **Alert Protocol** - Peers can alert other peers in the network about just detected IoC
* **File-sharing Protocol (FSP)** - File-sharing protocol for sharing bigger threat 
intelligence data. FSP gets inspiration from [IPFS](https://github.com/ipfs/ipfs) but
designs an access-control mechanism and dissemination mechanism to spread information about new files.
For more details see Architecture Overview section and the thesis itself.
* **Network's Opinion Protocol (NOP)** - NOP allows peers to ask other peers about their
opinion on a suspicious IoC. For more details see Architecture Overview section and the thesis itself.

### Auxiliary Protocols

* **Recommendation Protocol** - Recommendation Protocol is used by underlying Trust Model Fides. Fides 
requires opinions about newly connected peer from already-existing peers. Recommendation Protocol implements 
this feature. 
* **Peer-Query Protocol** - Peer-Query Protocol periodically asks connected peers about other peers so peers keep learning
about new peers in the system.

### Organisations 

We realize not all threat intelligence can be publicly disclosed for privacy and security reasons. Because of that
we propose a concept called Organisations to model trusted groups within the system. Iris
allows peers to address content just to members of given organisations. A peer can become a member of
an organisation by having its ID signed by organisation private key. Also, Fides Trust Model considers organisations in its
trust model when determining credibility of received data. For more details about theoretical definition and inner guts of Organisations,
see the [Iris](https://www.stratosphereips.org/thesis-projects-list/2022/3/12/global-permissionless-p2p-system-for-sharing-distributed-threat-intelligence)
and [Fides](https://www.stratosphereips.org/thesis-projects-list/2022/3/12/trust-model-for-global-peer-to-peer-intrusion-prevention-system) theses.

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

## Running

Starting a peer with reference configuration is as simple as running (assuming a Redis instance is running on local host):

> make run

### Configuration

Every peer needs a yaml configuration. To see all possible configuration fields, we refer a reader to see the source code of
[pkg/config/config.go](pkg/config/config.go) with all fields and explanatory comments.

```yaml
Identity:
  GenerateNewKey: true

Server:
  port: 9000
  Host: 127.0.0.1

Redis:
  Host: 127.0.0.1
  Tl2NlChannel: gp2p_tl2nl

Organisations:
  Trustworthy:
    - "12D3KooWErR8ZLhjAWYw4oj7gWLRPp99aupNU5HbFfVN9U12NBFZ"

PeerDiscovery:
  ListOfMultiAddresses:
    - "/ip4/127.0.0.1/udp/9001/quic 12D3KooWNxiCsZFyUFpLFNKDLEQDUK36my
       ifqufnnveK1jycMoJ8"
  DisableBootstrappingNodes: true
  ```
### Debug

After running a peer, it subscribes in a Redis channel to take commands from Slips (see 
explanation along with expected format of commands in [architecture.md](architecture.md).
To manually produce some of these commands in _redis-cli_, see [dev/redisobj.dev](dev/redisobj.dev).


## Tbd/Future work:
* automatic testing environment
* docker-compose with Slips, Fides and Iris
* signal handling with graceful shutdown
* search for buddies from the same organisation in peer discovery phase
* add message/bytes rate limitting per individual peers
* mechanism to purge keys in message cache after some time
* can people spoof QUIC sender? If not, tell Fides if stream is corrupted and cannot be deserialized so the peer can be punished
* responseStorage should not wait for responses from peers that disconnect. Otherwise when that happens it's gonna wait always till the timeout occurs
* wait in storageResponse only for responses from peers where requests were sucessfully sent (err was nil)
* report peers more when something wrong happens
* maybe delete file meta after expired elapsed? right now ElapsedAt is not used
* is basic manager really trimming peers based on their service trust? Check