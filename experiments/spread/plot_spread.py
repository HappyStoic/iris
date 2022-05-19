from spread import *
from matplotlib import pyplot as plt

FAVOR = reliability_favor_picker
FIRST = reliability_first_picker
RANDOM = random_picker


def strat(n: int, period: int, picker: Callable) -> SpreadScenario:
    if picker == FAVOR:
        d = f"({n}, {period}, C)"
    elif picker == FIRST:
        d = f"({n}, {period}, B)"
    elif picker == RANDOM:
        d = f"({n}, {period}, A)"
    else:
        d = "unknown"
    return SpreadScenario(spread_n_peers=n, spread_tick_interval=period, max_scenario_ticks=10000,
                          recipient_picker=picker, description=d)


def gen(networks: int = 200, max_peers=200, malicious_ratio: float = 0.0):
    networks_peers = random.randint(low=2, high=max_peers, size=networks)

    malicious_ratios = [0.0, 0.25, 0.5, 0.75]
    if malicious_ratio not in malicious_ratios:
        raise ValueError(f"wrong malicious ratio: {malicious_ratio}")

    malicious_mean_reliabilities = [0.0, 0.25, 0.5, 0.75]
    benign_mean_reliabilities = [0.25, 0.5, 0.75, 1.0]
    trust_view_accuracies = [0.0, 0.05, 0.15, 0.25]

    for i, n_peers in enumerate(networks_peers):
        network = generate_network(n_peers)

        # skipping not connected networks
        if not network.connected:
            continue

        m_idx = random.randint(low=0, high=len(malicious_mean_reliabilities))
        b_idx = random.randint(low=m_idx, high=len(benign_mean_reliabilities))
        malicious_rel = malicious_mean_reliabilities[m_idx]
        benign_rel = benign_mean_reliabilities[b_idx]
        trust_accuracy = random.choice(trust_view_accuracies)
        network.assign_reliabilities(malicious_ratio, malicious_rel, benign_rel, trust_accuracy)

        yield network


def plot_spread_messages(networks: int, malicious_ratio: float, strategies: List[SpreadScenario], file_pdf: str = ""):
    aggregated = {}
    for network in gen(networks=networks, max_peers=200, malicious_ratio=malicious_ratio):
        for s in strategies:
            key = s.description
            if key not in aggregated:
                aggregated[key] = [0] * 9999
            l = aggregated[key]

            res = evaluate(network, s, with_history=True)
            if res.end_tick == 9999:
                print("ERROR!")

            prev = 0
            aggr = []
            for tick in range(res.end_tick + 1):

                this_tick_msgs = res.msgs_history.get(tick, [])
                size_msgs_bening_peers = len([1 for (f,t) in this_tick_msgs if network.peers[t].is_benign])

                aggr.append(prev + size_msgs_bening_peers)
                prev = aggr[-1]

            for i in range(9999):
                val = aggr[-1]
                if i < len(aggr):
                    val = aggr[i]

                l[i] += val / networks

            aggregated[key] = l

    clean_aggr = {}
    fig, ax = plt.subplots()
    for k, l in aggregated.items():
        prev = None
        clean = {}
        clean_x = []
        clean_y = []
        for tick, val in enumerate(l):
            if val != prev:
                clean[tick] = val
                clean_x.append(tick)
                clean_y.append(val)
            prev = val
        clean_aggr[k] = clean

        ax.plot(clean_x, clean_y, label=k)
        ax.set_xlim(-200, 5000)
        ax.set_ylim(-10, 300),
        ax.set_xlabel("Ticks")
        ax.set_ylabel("Total number of messages sent" + (" to benign peers" if malicious_ratio != 0 else ""))
        ax.legend()

    if file_pdf:
        fig.savefig(file_pdf, format="pdf")
        print(f"stored fig to {file_pdf}")
    fig.show()
    print("done")


def plot_slow_spread_no_malicious():
    file = "/home/yourmother/moje/cvut/dp/figs/number-of-messages-slow-no-malicious.pdf"
    strategies = [strat(1, 100, FAVOR),
                  strat(1, 100, RANDOM),
                  strat(1, 100, FIRST),
                  strat(2, 250, FIRST),
                  strat(2, 250, FAVOR)
                  ]
    plot_spread_messages(networks=200, malicious_ratio=0.0, strategies=strategies, file_pdf=file)


def plot_slow_spread_malicious():
    map_strategies = {
        0.25: [strat(1, 100, RANDOM),
               strat(1, 100, FAVOR),
               strat(1, 100, FIRST),
               strat(2, 250, FAVOR),
               strat(2, 250, RANDOM)
               ],
        0.50: [strat(1, 100, RANDOM),
               strat(1, 100, FAVOR),
               strat(1, 100, FIRST),
               strat(2, 250, FAVOR),
               strat(2, 250, RANDOM)
               ],
        0.75: [strat(1, 50, FAVOR),
               strat(2, 100, FAVOR),
               strat(3, 250, FAVOR),
               strat(2, 100, FIRST),
               strat(1, 20, RANDOM)
               ],
    }
    malicious_ratio = 0.75
    file = f"/home/yourmother/moje/cvut/dp/figs/number-of-messages-slow-malicious{int(malicious_ratio*100)}.pdf"
    # file = ""
    s = map_strategies[malicious_ratio]
    plot_spread_messages(networks=200, malicious_ratio=malicious_ratio, strategies=s, file_pdf=file)


def plot_exponential_prob():
    xs = [x for x in np.arange(0, 1, 0.01)]
    a = 10
    f = lambda x: (pow(a,x)-1)/(a-1)
    ys = [f(x) for x in xs]

    fig, ax = plt.subplots()
    ax.plot(xs, ys, label="$y=\\frac{a^{x}-1}{a-1}, a=10$")
    ax.set_xlabel("service trust")
    ax.set_ylabel("probability")
    ax.legend()
    file = f"/home/yourmother/moje/cvut/dp/figs/exponential-prob.pdf"
    fig.savefig(file, format="pdf")
    print(f"stored res in {file}")
    fig.show()
    print("DONE")


if __name__ == "__main__":
    # plot_slow_spread_no_malicious()
    plot_slow_spread_malicious()
    # plot_exponential_prob()
