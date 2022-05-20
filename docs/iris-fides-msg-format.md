# Format of Messages exchanged between Iris and Fides through Redis channel

* TL - Trust Model (Fides)
* NL - Networking Layer (Iris)

## Generally

All messages are exchanged in one Redis channel. A structure of every message follows:
```yaml
{
"type": "...",
"version": 1,
"data": {}
}
```

Metada of peer object contains:
```yaml
{
  "id": "foo",
  "organisations":
            - "id"
}
```

## File Sharing Protocol

Initiated by TL

1.) TL sends information about starting sharing a file

severity has 2 purposes:
it affects how fast authorized peers will be notified about existence of the file
Slips can decide whether it wants for example only critical data
    description can contain w/e other needed object with data. For example
    size etc.
```yaml

{
"type": "tl2nl_file_share",
"version": 1,
"data": 
    "expired_at": <unix time until when offer the file>
    "severity": "CRITICAL"
    "rights": <list of organisations IDs or empty (all)
    "description": <optional blackbox metadata description for other instances>
    "path": <path on local filesystem where the file is located>
}

```
2.) NL provides TL metadata about available file
```yaml
{
"type": "nl2tl_file_share_received_metadata",
"version": 1,
"data":
    "file_id": <id>
    "severity": "CRITICAL"
    "sender": <Metadata of peer who has provided the file metadata>
    "description": <optional metadata description of file>
}
```

3.) TL wants to download a file
```yaml
{
"type": "tl2nl_file_share_download",
"version": 1,
"data":
    "file_id": <id>
}
```

4.) NL informs about successful file download
```yaml
 {
"type": "nl2tl_file_share_downloaded",
"version": 1,
"data": 
    "file_id": <id>
    "sender": <Metadata of peer who has provided the file>,
    "path": <path on local filesystem where the file is located>,
}
```

## Intelligence protocol

Initiated by TL

1.) TL wants to get thread intelligence about resource X from the closest network
```yaml
{
    "type": "tl2nl_intelligence_request"
    "version": 1,
    "data": 
        "payload": <blackbox for TL>
}
```

2.) NL forwards the alert to the network
```yaml
{
    "type": "nl2tl_intelligence_request",
    "version": 1,
    "data": 
        "request_id": <id>
        "sender": <Metadata of peer who's asking>
        "payload": <blackbox for TL>
}
```

3.) TL tells  to NL thread intelligence information
```yaml
{
    "type": "tl2nl_intelligence_response",
    "version": 1,
    "data": 
        "request_id": <id>
        "payload": <blackbox for TL>
}
```

4.) NL delivers threat intelligence information to TL
```yaml
{
    "type": "nl2tl_intelligence_response",
    "version": 1,
    "data": [
          { 
            "sender": <metadata of peer who's asking>
            "payload": <blackbox for TL>
          },
          ...,
    ]
}
```

## Alert protocol

Initiated by TL

1.) TL wants to notify the network about malicious resource X
```yaml
{
    "type": "tl2nl_alert",
    "version": 1,
    "data": 
    "payload": <blackbox for TL>
}
```

2.) NL forwards the alert to the network
```yaml
{
    "type": "nl2tl_alert",
    "version": 1,
    "data": 
        "sender": <Metadata of peer who's alerting>
        "payload": <blackbox for TL>
}
```

## Recommendation protocol


Initiated by TL

1.) TL wants recommendation from peers X about peer Y
```yaml
{
    "type": "tl2nl_recommendation_request",
    "version": 1,
    "data":
        "receiver_ids": <list of peer IDs X to ask>
        "payload": <blackbox for TL>
}
```

2.) NL asks TL about recommendation
```yaml
{
    "type": "nl2tl_recommendation_request",
    "version": 1,
    "data": 
        "request_id": <id>
        "sender": <Metadata of peer who's asking>
        "payload": <blackbox for TL>
}
```

3.) TL responds with recommendation
```yaml
{
    "type": "tl2nl_recommendation_response",
    "version": 1,
    "data":
        "request_id": <id of its matching redis request>,
        "recipient_id": <id of peer who was asking>,
        "payload": <blackbox for TL>,
}
```

4.) NL delivers reponse to TL
```yaml
{
    "type": "nl2tl_recommendation_response",
    "version": 1,
    "data": [
              {
                "sender": <Metadata of peer who's providing recommendation>,
                "payload": <blackbox for TL>,
              },
            ...,
    ]
}
```

## Report Peer's Missbehavior

Initiated by NL

1.) NL reports peer
```yaml
{
"type": "nl2tl_peer_report",
"version": 1,
"data": 
    "peer": <Metadata of peer who has provided the file metadata>
    "reason": <optional reason for logs>
}
```

## Peers Update Protocol

### Connected Peers Message

Initiated by NL. NL sends list of connected peers when there is a change. TL can use this information in Recommendation protocol

```yaml
{
"type": "nl2tl_peers_list",
"version": 1,
"data":
  "peers": <list of connected peers' Metadata>
}
```

### Peers Reliability Message

Initiated by TL. TL sends new reliability of peers
```yaml
{
"type": "tl2nl_peers_reliability",
"version": 1,
"data": [
          {
              "peer_id":    <id>,
              "reliability: <float>
          },
          ...
    ]
}
```
