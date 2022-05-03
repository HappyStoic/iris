import os
from collections import defaultdict
from dataclasses import dataclass, field
from typing import List, Dict, Set, Tuple, Callable

import multiprocessing as mp
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
    is_benign: bool = True


@dataclass(frozen=True)
class Network:
    peers: List[Peer]
    start: int
    connected: bool

    def assign_reliabilities(self, malicious_ratio: float, malicious_mean_rel: float, benign_mean_rel: float,
                             trust_accuracy: float):
        peers_total = len(self.peers)
        malicious_total = int(peers_total*malicious_ratio)
        benign_total = peers_total - malicious_total

        peer_indices = list(range(peers_total))
        random.shuffle(peer_indices)

        rels = random.normal(loc=[malicious_mean_rel]*malicious_total + [benign_mean_rel]*benign_total, scale=0.15)

        for i, (peer_idx, rel) in enumerate(zip(peer_indices, rels)):
            self.peers[peer_idx].is_benign = i >= malicious_total
            self.peers[peer_idx].rel = round_rel(rel)

        # this cannot be done in previous cycle because it needs rel to be set first
        self.__assign_random_rel_views(trust_accuracy)

    def __assign_random_rel_views(self, trust_accuracy: float):
        for peer in self.peers:
            rel_book = {}
            for (f, t) in peer.edges:
                # make "t" the other peer
                if t == peer.idx:
                    t = f
                rel_book[t] = random_rel_view(self.peers[t].rel, std_deviation=trust_accuracy)
            peer.rel_book = rel_book


@dataclass(frozen=True)
class SpreadScenario:
    spread_n_peers: int
    spread_tick_interval: int
    spread_total_ticks: int
    max_scenario_ticks: int
    recipient_picker: Callable[[Peer, List[int]], int]
    description: str


@dataclass(frozen=True)
class ScenarioResult:
    success: bool
    success_good_peers:  bool

    end_tick: int
    end_tick_good_peers: int

    repeated_msgs: int
    repeated_msgs_good_peers: int

    good_peers_notified: int
    unreliable_msgs: int
    spread_edges: Set[Tuple[int, int]]
    spread_ticks: Dict[Tuple[int, int], str]
    msgs_history: Dict[int, List[Tuple[int, int]]]


@dataclass(frozen=True)
class PeersParameters:
    malicious_ratio: float
    malicious_mean_reliability: float
    benign_mean_reliability: float
    trust_accuracy: float


@dataclass(frozen=True)
class NetworkResult:
    network_idx: int
    parameters: PeersParameters
    scenario_results: List[ScenarioResult]


def round_rel(rel: float) -> float:
    return 0.0 if rel < 0.0 else 1.0 if rel > 1.0 else rel


def random_rel_view(real_rel: float, std_deviation=0.1) -> float:
    return round_rel(random.normal(loc=real_rel, scale=std_deviation))


def generate_network(n_peers: int, verbose=False) -> Network:
    # generate peers
    peers = [Peer(idx=idx) for idx in range(n_peers)]

    # generate edges
    indices = list(range(n_peers))
    degrees = random.poisson(lam=7, size=n_peers)
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

    success, success_good_peers = False, False
    end_tick, end_tick_good_peers = 0, 0
    repeated_msgs, repeated_msgs_good_peers = 0, 0

    unreliable_msgs = 0
    spread_edges = set()
    spread_ticks = {}
    msgs_history = defaultdict(list)

    good_peers_notified = 0
    total_good_peers = len([None for p in network.peers if p.is_benign])

    for tick in range(scenario.max_scenario_ticks):
        end_tick = tick

        # read messages from inbox
        for (from_p, spread_to) in spread_inbox:
            msgs_history[tick].append((from_p, spread_to))
            if spread_to in received_peers:
                repeated_msgs += 1

                if network.peers[spread_to].is_benign:
                    repeated_msgs_good_peers += 1

                # we don't want to send msg to peers that told us that they already know about the msg
                try:
                    peers_possible_recipients[spread_to].remove(from_p)
                except ValueError:
                    pass  # we don't care... we just try to remove it

                continue

            received_peers.add(spread_to)
            peers_ticks_offsets[spread_to] = tick

            if network.peers[spread_to].is_benign:
                good_peers_notified += 1
                if good_peers_notified == total_good_peers:
                    success_good_peers = True
                    end_tick_good_peers = tick


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
            if peers_tick > scenario.spread_total_ticks:
                # timeout of ticks for this peer
                continue

            if peers_tick % scenario.spread_tick_interval != 0:
                # this peer is not sending msgs in this tick
                continue

            # throw dices for mocking reliability
            dices = random.rand(min(scenario.spread_n_peers, len(peers_possible_recipients[p])))
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
                         success_good_peers=success_good_peers,

                         end_tick=end_tick,
                         end_tick_good_peers=end_tick_good_peers,

                         repeated_msgs=repeated_msgs,
                         repeated_msgs_good_peers=repeated_msgs_good_peers,

                         good_peers_notified=good_peers_notified,
                         unreliable_msgs=unreliable_msgs,
                         spread_edges=spread_edges,
                         spread_ticks=spread_ticks,
                         msgs_history=msgs_history)
    return res


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


def run(testing_scenarios: List[SpreadScenario], max_peers=50, n_networks=10) -> (List[Network], Dict[float, List[NetworkResult]]):
    networks_peers = random.randint(low=2, high=max_peers, size=n_networks)

    malicious_ratios = [0.0, 0.25, 0.5, 0.75]
    malicious_mean_reliabilities = [0.0, 0.25, 0.5, 0.75]
    benign_mean_reliabilities = [0.25, 0.5, 0.75, 1.0]
    trust_view_accuracies = [0.0, 0.05, 0.15, 0.25]

    networks_cnt = 0
    networks = []
    results = defaultdict(list)
    for i, n_peers in enumerate(networks_peers):
        network = generate_network(n_peers)

        # skipping not connected networks
        if not network.connected:
            continue

        for malicious_ratio in malicious_ratios:

            m_idx = random.randint(low=0, high=len(malicious_mean_reliabilities))
            b_idx = random.randint(low=m_idx, high=len(benign_mean_reliabilities))
            malicious_rel = malicious_mean_reliabilities[m_idx]
            benign_rel = benign_mean_reliabilities[b_idx]
            trust_accuracy = random.choice(trust_view_accuracies)

            parameters = PeersParameters(malicious_ratio=malicious_ratio, benign_mean_reliability=benign_rel,
                                         malicious_mean_reliability=malicious_rel, trust_accuracy=trust_accuracy)
            # print("doing parameters ", parameters)

            network.assign_reliabilities(malicious_ratio, malicious_rel, benign_rel, trust_accuracy)
            scenario_results = [evaluate(network, scenario) for scenario in testing_scenarios]
            results[malicious_ratio].append(NetworkResult(network_idx=networks_cnt, parameters=parameters,
                                                          scenario_results=scenario_results))

        #     plot(network, scenario_results[0])
        #     break
        # break

        networks.append(network)
        networks_cnt += 1

        if networks_cnt % 5 == 0:
            print(f"{networks_cnt}th network done and evaluated")

    print(f"Done... {networks_cnt}th network done and evaluated")
    return networks, results


def all_pickers_scenario(num_peers: int, each_tick: int, total_ticks: int, max_scenario_ticks=10000) -> List[SpreadScenario]:
    txt = f"spreading up to {num_peers} peers every {each_tick} tick for maximum {total_ticks} ticks with "
    return [
        SpreadScenario(spread_n_peers=num_peers, spread_tick_interval=each_tick, spread_total_ticks=total_ticks,
                       max_scenario_ticks=max_scenario_ticks, recipient_picker=reliability_first_picker,
                       description=f"{txt} reliability first picker"),
        SpreadScenario(spread_n_peers=num_peers, spread_tick_interval=each_tick, spread_total_ticks=total_ticks,
                       max_scenario_ticks=max_scenario_ticks, recipient_picker=reliability_favor_picker,
                       description=f"{txt} reliability favor picker"),
        SpreadScenario(spread_n_peers=num_peers, spread_tick_interval=each_tick, spread_total_ticks=total_ticks,
                       max_scenario_ticks=max_scenario_ticks, recipient_picker=random_picker,
                       description=f"{txt} random picker")
    ]


def get_scenarios() -> List[SpreadScenario]:
    ALL = 99999

    total_ticks_opts = [1, 5, 100, ALL]
    each_tick_opts = [1, 2, 4, 10, 20, 50, 100, 500]
    num_peers_opts = [1, 2, 3, 5, 7, 9, ALL]
    s = []
    for total_ticks in total_ticks_opts:
        for each_tick in each_tick_opts:
            if each_tick > total_ticks:
                continue
            for num_peers in num_peers_opts:
                s += all_pickers_scenario(num_peers=num_peers, each_tick=each_tick, total_ticks=total_ticks)
    return s


def run_process(q: mp.Queue, n_networks, testing_scenarios: List[SpreadScenario], max_peers=50):
    random.seed(os.getpid())
    networks, rel_results = run(testing_scenarios, n_networks=n_networks, max_peers=max_peers)
    q.put((networks, rel_results))


def test_multiprocess(scenarios: List[SpreadScenario], total_networks : int = 50, n_procs: int = 5, max_peers=50) -> Tuple[List[Network], List[NetworkResult]]:
    print("running multiprocess test")

    process_networks = total_networks // n_procs
    q = mp.Queue()
    networks, results = [], defaultdict(list)
    procs = []
    for i in range(n_procs):
        p = mp.Process(target=run_process, args=(q, process_networks, scenarios, max_peers))
        print(f"process {i} started")
        p.start()
        procs.append(p)

    for _ in range(n_procs):
        sub_networks, sub_results = q.get()
        networks += sub_networks
        for k, sub_v in sub_results.items():
            results[k] += sub_v

    for i, p in enumerate(procs):
        p.join()
        print(f"process {i} finished")

    print("done")
    print(f"total networks received {len(networks)}")
    return networks, results


def test(scenarios: List[SpreadScenario]):
    networks, rel_results = run(scenarios, n_networks=5)


def save_results(exp_name: str, data: Tuple[List[Network], List[NetworkResult], List[SpreadScenario]]):
    import pickle

    file = f"{exp_name}.pickle"
    with open(file, 'wb') as handle:
        pickle.dump(data, handle, protocol=pickle.HIGHEST_PROTOCOL)

    print(f"saved result into {file}")


if __name__ == "__main__":
    save = True
    experiment = "exp-big1"
    scenarios = get_scenarios()
    total_networks = 100
    max_peers = 100
    n_procs = 7

    # test(scenarios=scenarios)
    networks, results = test_multiprocess(scenarios=scenarios, total_networks=total_networks, n_procs=n_procs, max_peers=max_peers)

    if save:
        save_results(experiment, (networks, results, scenarios))
    print("the end")
