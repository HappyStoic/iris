from collections import defaultdict
from dataclasses import dataclass
from typing import List, Dict, Set, Tuple, Callable

import numpy as np
import pandas as pd
from jaal import Jaal
from numpy import random


@dataclass
class Peer:
    idx: int
    edges: Set[Tuple[int, int]]
    rel_book: Dict[int, float]
    rel: float


@dataclass(frozen=True)
class Network:
    peers: List[Peer]
    start: int
    reliability: float
    connected: bool


@dataclass(frozen=True)
class SpreadScenario:
    num_peers: int
    each_tick: int
    total_ticks: int
    recipient_picker: Callable[[List[Peer]], Peer]
    description: str


@dataclass(frozen=True)
class Result:
    success: bool
    end_tick: int
    spam_msgs: int
    success_msgs: int
    unreliable_msgs: int
    spread_edges: Set[Tuple[int, int]]
    spread_ticks: Dict[Tuple[int, int], int]
    msgs_history: Dict[int, List[Tuple[int, int]]]
    tick_of_25percentile_spread: int
    tick_of_50percentile_spread: int
    tick_of_75percentile_spread: int


def random_rel_view(real_rel: float) -> float:
    rel = random.normal(loc=real_rel, scale=10)
    return 0 if rel < 0 else 100 if rel > 100 else rel


def generate_network(n_peers: int, network_rel: int, network_degree: int) -> Network:
    # first for every peer calculate its reliability and degree so overall average first rel arg
    rels = random.normal(loc=network_rel, scale=40, size=n_peers)
    degrees = random.poisson(lam=network_degree, size=n_peers)

    # generate peers
    peers = []
    for idx, rel in enumerate(rels):
        rel = 100 if rel > 100 else 1 if rel < 1 else rel
        peers.append(Peer(idx=idx, edges=set(), rel_book=defaultdict(float), rel=rel))

    # generate edges and reliability views
    indices = list(range(n_peers))
    for from_peer, degree in zip(peers, degrees):

        # poisson distribution can give us 0, we want at least one connection
        if degree == 0:
            degree = 1

        random.shuffle(indices)
        done = len(from_peer.edges)
        # find connections
        for to_peer_idx in indices:
            if done >= degree:
                break

            if to_peer_idx == from_peer.idx:
                continue  # we cannot have a connection to itself

            edge = (from_peer.idx, to_peer_idx) if from_peer.idx < to_peer_idx else (to_peer_idx, from_peer.idx)

            if edge in from_peer.edges:
                continue  # this edge already exists (targeted node connected to this node)

            peers[to_peer_idx].edges.add(edge)
            peers[to_peer_idx].rel_book[from_peer.idx] = random_rel_view(from_peer.rel)

            from_peer.edges.add(edge)
            from_peer.rel_book[to_peer_idx] = random_rel_view(peers[to_peer_idx].rel)

            done += 1

    # generate starting peer
    start_peer = random.randint(0, n_peers)

    # check by DFS if graph is connected
    visited = {start_peer}
    next = list(peers[start_peer].edges)
    for e in next:
        first, second = e

        if first not in visited:
            visited.add(first)
            next += list(peers[first].edges)

        if second not in visited:
            visited.add(second)
            next += list(peers[second].edges)
    connected = len(visited) == n_peers

    return Network(peers=peers, start=start_peer, connected=connected, reliability=network_rel)


def plot(network: Network, result: Result):
    peers = network.peers
    start = network.start

    # create nodes dataframe
    indices = list(range(len(peers)))
    roles = ["none" for _ in indices]
    roles[start] = "start"
    nodes_df = pd.DataFrame(zip(indices, roles), roles, columns=["id", "role"])

    # create edges dataframe
    rows = set()
    for p in peers:
        new_rows = {(f, t, "yes" if (f, t) in result.spread_edges else "no",
                     result.spread_ticks.get((f, t), "")) for (f, t) in p.edges}
        rows = rows.union(new_rows)

    edges_df = pd.DataFrame(rows, columns=["from", "to", "is_spread_edge", "label"])

    print("plotting ", result)

    # init Jaal and run server
    Jaal(edges_df, nodes_df).plot()


def evaluate(network: Network, scenario: SpreadScenario) -> Result:
    spread_inbox = {(None, network.start)}
    received_peers = set()
    peers_ticks_offsets = [None for _ in network.peers]
    peers_possible_recipients = [[network.peers[f] if f != p.idx else network.peers[t] for (f, t) in p.edges] for p in network.peers]

    percentile25, percentile50, percentile75 = None, None, None
    spam_msgs = 0
    success_msgs = 0
    unreliable_msgs = 0
    msgs_history = defaultdict(list)
    success = False
    spread_edges = set()
    spread_ticks = {}

    quarter_peers = len(network.peers) / 4
    half_peers = len(network.peers) / 2
    three_quarter_peers = 3*quarter_peers

    for tick in range(MAX_TICKS):
        # read messages from inbox
        for (from_p, spread_to) in spread_inbox:
            msgs_history[tick].append((from_p, spread_to))
            if spread_to in received_peers:
                spam_msgs += 1
                continue

            success_msgs += 1
            received_peers.add(spread_to)
            peers_ticks_offsets[spread_to] = tick

            if not percentile25 and len(received_peers) > quarter_peers:
                percentile25 = tick
            if not percentile50 and len(received_peers) > half_peers:
                percentile50 = tick
            if not percentile75 and len(received_peers) > three_quarter_peers:
                percentile75 = tick

            if from_p is not None:
                peers_possible_recipients[spread_to].remove(network.peers[from_p])

                e = (from_p, spread_to) if from_p < spread_to else (spread_to, from_p)
                spread_edges.add(e)
                spread_ticks[e] = f"tick {tick}"

        # does already everyone know about the message?
        if len(received_peers) == len(network.peers):
            success = True
            break

        # clean inbox for next tick
        spread_inbox = set()

        # send messages
        for p in received_peers:
            peers_tick = tick - peers_ticks_offsets[p]
            if peers_tick > scenario.total_ticks:
                # timeout of ticks  for this peer
                continue

            if peers_tick % scenario.each_tick != 0:
                # this peer is not sending msgs in this tick
                continue

            # throw dices for mocking reliability
            dices = random.rand(scenario.num_peers) * 100
            for roll in dices:
                if p != network.start and roll > network.peers[p].rel:
                    # reliability failure (starting node does never fail)
                    unreliable_msgs += 1
                    continue

                if not peers_possible_recipients[p]:
                    break

                send_to = scenario.recipient_picker(peers_possible_recipients[p])
                peers_possible_recipients[p].remove(send_to)

                spread_inbox.add((p, send_to.idx))

    res = Result(success=success,
                 end_tick=tick,
                 spam_msgs=spam_msgs,
                 success_msgs=success_msgs,
                 unreliable_msgs=unreliable_msgs,
                 msgs_history=msgs_history,
                 spread_edges=spread_edges,
                 tick_of_25percentile_spread=percentile25,
                 tick_of_50percentile_spread=percentile50,
                 tick_of_75percentile_spread=percentile75,
                 spread_ticks=spread_ticks)
    return res


NUMBER_OF_NETWORKS = 2000
MAX_TICKS = 10000


def reliability_picker(peers: List[Peer]) -> Peer:
    return max(peers, key=lambda peer: peer.rel)


def reliability_favor_picker(peers: List[Peer]) -> Peer:
    a = 10
    rand_rel = 1 - ((a**random.rand()) - 1) / (a - 1)
    peer = peers[0]
    dist = np.abs(peer.rel - rand_rel)
    for p in peers[1:]:
        cur_dist = np.abs(peer.rel - rand_rel)
        if cur_dist < dist:
            dist = cur_dist
            peer = p
    return peer


def run():
    networks_peers = random.randint(low=2, high=50, size=NUMBER_OF_NETWORKS)
    networks_rels = random.randint(low=5, high=101, size=NUMBER_OF_NETWORKS)
    networks_degrees = random.randint(low=1, high=(networks_peers // 2) + 1, size=NUMBER_OF_NETWORKS)

    random_picker = lambda peers: random.choice(peers)
    scenarios = [
        SpreadScenario(num_peers=2, each_tick=10, total_ticks=100, recipient_picker=random_picker,
                       description="random picker"),
        SpreadScenario(num_peers=2, each_tick=10, total_ticks=100, recipient_picker=reliability_favor_picker,
                       description="reliability picker"),
        # SpreadScenario(num_peers=5, each_tick=10, total_ticks=100, recipient_picker=random_picker,
        #                description="random picker"),
        # SpreadScenario(num_peers=5, each_tick=5, total_ticks=100, recipient_picker=random_picker,
        #                description="random picker"),
        # SpreadScenario(num_peers=2, each_tick=3, total_ticks=100, recipient_picker=random_picker,
        #                description="random picker"),
    ]
    networks_results = []
    a, a_total, a_spam = 0, 0, 0
    b, b_total, b_spam = 0, 0, 0
    for n_peers, network_rel, network_degree in zip(networks_peers, networks_rels, networks_degrees):
        network = generate_network(n_peers, network_rel, network_degree)
        results = [evaluate(network, scenario) for scenario in scenarios]
        networks_results.append((network, results))

        if results[0].success:
            a += results[0].end_tick
            a_total += 1
            a_spam += results[0].spam_msgs
        if results[1].success:
            b += results[1].end_tick
            b_total += 1
            b_spam += results[1].spam_msgs

        # print(f"network of {len(network.peers)} peers with start at {network.start} "
        #       f"| connected {network.connected} | reliability {network.reliability}")
        print(f"network {len(networks_results)} done and evaluated")

        plot(network, results[0])
        break
    print(a, a_total, a/a_total, a_spam)
    print(b, b_total, b/b_total, b_spam)


if __name__ == "__main__":
    run()
