# Architecture, Implementation details

First, let me mention that taxonomy in the code-base slightly differs from the thesis as both parts
went through some progress. That's why, thesis talks about `service trust`, code-base refers to it as `reliability`. Also,
thesis talks about `Network Opinion Protocol`, code-base refers to it as `Intelligence Protocol`.

Iris is implemented in Go using LibP2P [LibP2P project](https://github.com/libp2p).
Check the following diagram to see division of responsibilities:

![Responsibilities Diagram](media/responsibilities-diagram.jpg)

### Components in the Assumed System

* **Iris P2P System** - current repository
* [**Slips Intrusion Prevention System**](https://github.com/draliii/StratosphereLinuxIPS) - Slips (Python) is a central brain of all the detections. Slips uses Iris as a tool to communicate with other Slips instances
* [**Fides Trust Model**](https://github.com/lukasforst/fides) - Fides (Python) is a Trust Model designed simultaneously with Iris for highly adversarial networks. Fides is implemented as built-in module inside Slips.

See a diagram that depicts communication flow between the components:

![Communication Diagram](media/components-communication-diagram.jpg)

Since Iris is implemented in Go and Fides is implemented in Python, only Fides directly interacts with Slips. Iris exchanges
messages only with Fides through Redis channel and with other peers. To see the defined message structure between Fides and Iris, see [iris-fides-msg-format.md](iris-fides-msg-format.md) file.

### Networking

As a transport layer Iris uses QUIC protocol (which runs on top of UDP)

### Core Protocols

Iris proposes and implements 3 core protocols to exchange threat intelligence data:
* Alert Protocol
* Intelligence Protocol (or Network Opinion Protocol)
* File Sharing Protocol

For details about these protocols we refer reader to see the text of the thesis.

### Auxiliary Behaviour/Protocols
#### Peer Query Protocol

TBD
* **Peer-Query Protocol** - Peer-Query Protocol periodically asks connected peers about other peers so peers keep learning
  about new peers in the system.

#### Recommendation Protocol

TBD
* **Recommendation Protocol** - Recommendation Protocol is used by underlying Trust Model Fides. Fides
  requires opinions about newly connected peer from already-existing peers. Recommendation Protocol implements
  this feature.

#### Connection Manager

TBD

#### Organisation Member Updater

TBD

### Peer Configuration

Iris requires a yaml configuration to run a peer. For all possible configuration fields, we refer a reader to see the source code of
[pkg/config/config.go](pkg/config/config.go) with all up-to-date fields. A small working example of the most important fields can be seen below:
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