import time
from collections import defaultdict
from dataclasses import dataclass, field
from typing import List, Dict, Set, Tuple, Callable

import numpy as np
import pandas as pd
from jaal import Jaal
from numpy import random


@dataclass
class Peer:
    idx: int
    edges: Set[Tuple[int, int]] = field(default_factory=set)
    rel_book: Dict[int, float] = field(default_factory=dict)
    rel: float = -1.0

    def is_good(self) -> bool:
        return self.rel > GOOD_PEER_REL_THRESHOLD


@dataclass(frozen=True)
class Network:
    peers: List[Peer]
    start: int
    connected: bool

    def assign_normal_reliabilities(self, avg_rel: float):
        reliabilities = random.normal(loc=avg_rel, scale=10, size=len(self.peers))

        for peer, rel in zip(self.peers, reliabilities):
            peer.rel = rel

        # this cannot be done in previous cycle because it needs rel to be set first
        self.__assign_random_rel_views()

    def assign_malicious_rate(self, rate: float):
        peers_total = len(self.peers)
        malicious_total = int(peers_total*rate)

        peer_indices = list(range(peers_total))
        random.shuffle(peer_indices)

        for i, peer_idx in enumerate(peer_indices):
            self.peers[peer_idx].rel = 0.0 if i < malicious_total else 100.0

        # this cannot be done in previous cycle because it needs rel to be set first
        self.__assign_random_rel_views()

    def __assign_random_rel_views(self):
        for peer in self.peers:
            rel_book = {}
            for (f, t) in peer.edges:
                # make "t" the other peer
                if t == peer.idx:
                    t = f
                rel_book[t] = random_rel_view(self.peers[t].rel)
            peer.rel_book = rel_book


@dataclass(frozen=True)
class SpreadScenario:
    num_peers: int
    each_tick: int
    total_ticks: int
    recipient_picker: Callable[[Peer, List[int]], int]
    description: str


@dataclass(frozen=True)
class ScenarioResult:
    success: bool
    end_tick: int
    spam_msgs: int
    spam_msgs_good_peers: int
    success_msgs: int
    unreliable_msgs: int
    spread_edges: Set[Tuple[int, int]]
    spread_ticks: Dict[Tuple[int, int], str]
    msgs_history: Dict[int, List[Tuple[int, int]]]
    tick_of_25_spread: int
    tick_of_50_spread: int
    tick_of_75_spread: int
    tick_of_95_spread: int
    tick_of_25_good_spread: int
    tick_of_50_good_spread: int
    tick_of_75_good_spread: int
    tick_of_95_good_spread: int
    good_peers_in_network: int
    good_peers_notified: int


@dataclass(frozen=True)
class NetworkResult:
    network_idx: int
    scenario_results: List[ScenarioResult]


def random_rel_view(real_rel: float, std_deviation=10) -> float:
    rel = random.normal(loc=real_rel, scale=std_deviation)
    return 0 if rel < 0 else 100 if rel > 100 else rel


def generate_network(n_peers: int, network_degree: int, verbose=False) -> Network:
    # generate peers
    peers = [Peer(idx=idx) for idx in range(n_peers)]

    # generate edges
    indices = list(range(n_peers))
    degrees = random.poisson(lam=network_degree, size=n_peers)
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
            from_peer.edges.add(edge)

            done += 1

    # generate starting peer
    start_peer = random.randint(0, n_peers)

    # check by DFS if graph is connected
    visited = {start_peer}
    next_edges = list(peers[start_peer].edges)
    connected = False
    for e in next_edges:
        first, second = e

        if first not in visited:
            visited.add(first)
            next_edges += list(peers[first].edges)

        if second not in visited:
            visited.add(second)
            next_edges += list(peers[second].edges)

        connected = len(visited) == n_peers
        if connected:
            break

    if verbose:
        print(f"generated network with {len(peers)} peers, start in {start_peer}, connected {connected}")
    return Network(peers=peers, start=start_peer, connected=connected)


def plot(network: Network, result: ScenarioResult):
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


# Possible candidates for recipients:
#       * I have not sent the msg to this peer yet
#       * This peer did not send me the msg at any point in the history
def evaluate(network: Network, scenario: SpreadScenario) -> ScenarioResult:
    spread_inbox = {(None, network.start)}
    received_peers = set()
    peers_ticks_offsets = [None for _ in network.peers]
    peers_possible_recipients: List[List[int]] = [[f if f != p.idx else t for (f, t) in p.edges] for p in network.peers]

    spread25, spread50, spread75, spread95 = None, None, None, None
    spread25_good_peers, spread50_good_peers, spread75_good_peers, spread95_good_peers = None, None, None, None
    spam_msgs = 0
    spam_msgs_good_peers = 0
    success_msgs = 0
    unreliable_msgs = 0
    msgs_history = defaultdict(list)
    success = False
    spread_edges = set()
    spread_ticks = {}
    good_peers_notified = 0

    total_good_peers = len([None for p in network.peers if p.is_good()])
    quarter_good_peers = total_good_peers / 4
    half_good_peers = total_good_peers / 2
    three_quarter_good_peers = 3 * quarter_good_peers
    perc95_good_peers = 95 * (total_good_peers / 100)

    quarter_peers = len(network.peers) / 4
    half_peers = len(network.peers) / 2
    three_quarter_peers = 3 * quarter_peers
    perc95_peers = 95 * len(network.peers) / 100

    for tick in range(MAX_TICKS):
        # read messages from inbox
        for (from_p, spread_to) in spread_inbox:
            msgs_history[tick].append((from_p, spread_to))
            if spread_to in received_peers:
                spam_msgs += 1

                if network.peers[spread_to].is_good():
                    spam_msgs_good_peers += 1

                # we don't want to send msg to peers that told us that they already know about the msg
                try:
                    peers_possible_recipients[spread_to].remove(from_p)
                except ValueError:
                    pass  # we don't care... we just try to remove it

                continue

            success_msgs += 1
            received_peers.add(spread_to)
            peers_ticks_offsets[spread_to] = tick

            if network.peers[spread_to].is_good():
                good_peers_notified += 1
                if not spread25_good_peers and good_peers_notified > quarter_good_peers:
                    spread25_good_peers = tick
                if not spread50_good_peers and good_peers_notified > half_good_peers:
                    spread50_good_peers = tick
                if not spread75_good_peers and good_peers_notified > three_quarter_good_peers:
                    spread75_good_peers = tick
                if not spread95_good_peers and good_peers_notified > perc95_good_peers:
                    spread95_good_peers = tick

            if not spread25 and len(received_peers) > quarter_peers:
                spread25 = tick
            if not spread50 and len(received_peers) > half_peers:
                spread50 = tick
            if not spread75 and len(received_peers) > three_quarter_peers:
                spread75 = tick
            if not spread95 and len(received_peers) > perc95_peers:
                spread95 = tick

            if from_p is not None:
                peers_possible_recipients[spread_to].remove(from_p)

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
                # timeout of ticks for this peer
                continue

            if peers_tick % scenario.each_tick != 0:
                # this peer is not sending msgs in this tick
                continue

            # throw dices for mocking reliability
            dices = random.rand(min(scenario.num_peers, len(peers_possible_recipients[p]))) * 100
            for roll in dices:
                if not peers_possible_recipients[p]:
                    break

                if p != network.start and roll > network.peers[p].rel:
                    # reliability failure (starting node does never fail)
                    unreliable_msgs += 1
                    continue

                send_to = scenario.recipient_picker(network.peers[p], peers_possible_recipients[p])
                peers_possible_recipients[p].remove(send_to)

                spread_inbox.add((p, send_to))

    res = ScenarioResult(success=success,
                         end_tick=tick,
                         spam_msgs=spam_msgs,
                         spam_msgs_good_peers=spam_msgs_good_peers,
                         success_msgs=success_msgs,
                         unreliable_msgs=unreliable_msgs,
                         msgs_history=msgs_history,
                         spread_edges=spread_edges,
                         tick_of_25_spread=spread25,
                         tick_of_50_spread=spread50,
                         tick_of_75_spread=spread75,
                         tick_of_95_spread=spread95,
                         tick_of_25_good_spread=spread25_good_peers,
                         tick_of_50_good_spread=spread50_good_peers,
                         tick_of_75_good_spread=spread75_good_peers,
                         tick_of_95_good_spread=spread95_good_peers,
                         good_peers_in_network=total_good_peers,
                         good_peers_notified=good_peers_notified,
                         spread_ticks=spread_ticks, )
    return res


GOOD_PEER_REL_THRESHOLD = 75
MAX_TICKS = 10000


def reliability_first_picker(sender: Peer, recipients: List[int]) -> int:
    return max(recipients, key=lambda idx: sender.rel_book[idx])


def reliability_favor_picker(sender: Peer, recipients: List[int]) -> int:
    rel_book = sender.rel_book
    a = 10

    rand_rel = 1 - ((a ** random.rand()) - 1) / (a - 1)
    best_peer = recipients[0]
    best_dist = np.abs(rel_book[best_peer] - rand_rel)
    for candidate in recipients[1:]:
        cur_dist = np.abs(rel_book[candidate] - rand_rel)
        if cur_dist < best_dist:
            best_dist = cur_dist
            best_peer = candidate

    return best_peer


def random_picker(_: Peer, recipients: List[int]) -> int:
    return random.choice(recipients)


def run(testing_scenarios: List[SpreadScenario], n_networks=10) -> (List[Network], Dict[str, List[NetworkResult]]):
    networks_peers = random.randint(low=2, high=50, size=n_networks)
    networks_degrees = random.randint(low=1, high=(networks_peers // 2) + 1, size=n_networks)

    testing_normal_reliabilities = [20.0, 40.0, 60.0, 80.0]
    testing_malicious_rates = [0.0, 0.25, 0.50, 0.75]

    networks_cnt = 0
    networks = []
    results = defaultdict(list)
    for i, (n_peers, network_degree) in enumerate(zip(networks_peers, networks_degrees)):
        network = generate_network(n_peers, network_degree)

        # skipping not connected networks
        if not network.connected:
            continue

        for normal_rel in testing_normal_reliabilities:
            name = f"normal distribution of reliability with mean {normal_rel}"

            network.assign_normal_reliabilities(normal_rel)
            scenario_results = [evaluate(network, scenario) for scenario in testing_scenarios]

            results[name].append(NetworkResult(network_idx=networks_cnt, scenario_results=scenario_results))

        #     plot(network, scenario_results[-1])
        #     break
        # break

        for malicious_rate in testing_malicious_rates:
            name = f"{malicious_rate}% of malicious peers in the network"

            network.assign_malicious_rate(malicious_rate)
            scenario_results = [evaluate(network, scenario) for scenario in testing_scenarios]

            results[name].append(NetworkResult(network_idx=networks_cnt, scenario_results=scenario_results))

        networks.append(network)
        networks_cnt += 1
        if networks_cnt % 20 == 0:
            print(f"{networks_cnt}th network done and evaluated")

    print(f"Finished, {networks_cnt}th network done and evaluated")
    return networks, results


def new_scenarios(num_peers: int, each_tick: int, total_ticks: int) -> List[SpreadScenario]:
    txt = f"spreading up to {num_peers} peers every {each_tick} tick for maximum {total_ticks} ticks with "
    return [
        SpreadScenario(num_peers=num_peers, each_tick=each_tick, total_ticks=total_ticks,
                       recipient_picker=reliability_first_picker, description=f"{txt} reliability first picker"),
        SpreadScenario(num_peers=num_peers, each_tick=each_tick, total_ticks=total_ticks,
                       recipient_picker=reliability_favor_picker, description=f"{txt} reliability favor picker"),
        SpreadScenario(num_peers=num_peers, each_tick=each_tick, total_ticks=total_ticks,
                       recipient_picker=random_picker, description=f"{txt} random picker")
    ]


def test():
    testing_scenarios = new_scenarios(num_peers=2, each_tick=10, total_ticks=100)
    testing_scenarios += new_scenarios(num_peers=2, each_tick=5, total_ticks=100)
    testing_scenarios += new_scenarios(num_peers=5, each_tick=100, total_ticks=500)
    testing_scenarios += new_scenarios(num_peers=99, each_tick=10, total_ticks=50)
    testing_scenarios += new_scenarios(num_peers=99, each_tick=1, total_ticks=0)

    networks, rel_results = run(testing_scenarios, n_networks=10)


if __name__ == "__main__":
    test()
