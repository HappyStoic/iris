import json
from pathlib import Path

import networkx as nx
import plotly.graph_objects as go
from dash import Dash, dcc, html, Input, Output
from dateutil import parser as date_parser

EXP_DIR = "out/17:36-Mar22"


def load_logs() -> (int, list):
    d = Path(EXP_DIR)
    outs = sorted([f for f in d.glob('ins*/out') if f.is_file()], key=lambda x: int(x.parent.name[3:]))

    ids_to_ins = {}
    logs = []
    for ins, outf in enumerate(outs):
        with open(outf, "r") as f:
            loglines = f.readlines()
        logs.append([json.loads(line) for line in loglines])

        for line in logs[-1]:
            if "created node with ID:" in line["msg"]:
                peer_id = line["msg"].split(" ")[-1]
                ids_to_ins[peer_id] = ins
                break

    actions = []
    for ins, logs in enumerate(logs):
        for line in logs:
            msg = line["msg"]

            if not ("connected" in msg or "disconnected" in msg):
                continue

            ts = date_parser.parse(line["ts"])

            action = 1  # add
            if "disconnected" in msg:
                action = 0

            target = msg.split(" ")[2].strip("'")
            target_ins = ids_to_ins[target]
            actions.append((action, (ins, target_ins), ts))

    # sort actions based on timestamp
    actions.sort(key=lambda x: x[2])

    return len(outs), actions


def load_edges(actions: list, upto: int) -> set:
    edges = set()

    for count, action in enumerate(actions):
        operation, edge, _ = action
        if operation == 1:
            edges.add(edge)
        else:
            edges.discard(edge)

        if count >= upto:
            break

    return edges


def load_graph(n: int, actions: list, upto: int = -1) -> nx.Graph:
    # construct graph
    graph = nx.Graph()
    for i in range(n):
        graph.add_node(i)

    edges = load_edges(actions, upto)
    graph.add_edges_from(edges)
    return graph


def set_layout(graph: nx.Graph, _: str):
    # pos = nx.circular_layout(graph)
    pos = nx.spring_layout(graph, k=0.5, iterations=50)
    # pos = nx.spring_layout(G, k=0.5, iterations=50)
    # pos = nx.spring_layout(G, k=0.5, iterations=50)
    # pos = nx.spring_layout(G, k=0.5, iterations=50)
    # pos = nx.spring_layout(G, k=0.5, iterations=50)

    for node, p in pos.items():
        graph.nodes[node]['pos'] = p


def get_fig(G: nx.Graph):
    edge_trace = go.Scatter(
        name="edge_trace",
        x=[],
        y=[],
        line=dict(width=0.5, color='#888'),
        hoverinfo='none',
        mode='lines')

    for edge in G.edges():
        x0, y0 = G.nodes[edge[0]]['pos']
        x1, y1 = G.nodes[edge[1]]['pos']
        edge_trace['x'] += tuple([x0, x1, None])
        edge_trace['y'] += tuple([y0, y1, None])

    node_trace = go.Scatter(
        x=[],
        y=[],
        text=[],
        mode='markers',
        hoverinfo='text',
        marker=dict(
            showscale=True,
            colorscale='RdBu',
            reversescale=True,
            color=[],
            size=15,
            colorbar=dict(
                thickness=10,
                title='Node Connections',
                xanchor='left',
                titleside='right'
            ),
            line=dict(width=0)))

    for node in G.nodes():
        x, y = G.nodes[node]['pos']
        node_trace['x'] += tuple([x])
        node_trace['y'] += tuple([y])

    # for node, adjacencies in enumerate(G.adjacency()):
    #     node_trace['marker']['color'] += tuple([len(adjacencies[1])])
    #     node_info = adjacencies[0] + ' # of connections: ' + str(len(adjacencies[1]))
    #     node_trace['text'] += tuple([node_info])

    fig = go.Figure(data=[edge_trace, node_trace],
                    layout=go.Layout(
                        title='<br>AT&T network connections',
                        titlefont=dict(size=16),
                        showlegend=False,
                        hovermode='closest',
                        margin=dict(b=20, l=5, r=5, t=40),
                        annotations=[dict(
                            text="No. of connections",
                            showarrow=False,
                            xref="paper", yref="paper")],
                        xaxis=dict(showgrid=False, zeroline=False, showticklabels=False),
                        yaxis=dict(showgrid=False, zeroline=False, showticklabels=False)))

    return fig


def update_fig_edges(graph, fig):
    print("update in")

    xs = []
    ys = []
    for edge in graph.edges():
        x0, y0 = graph.nodes[edge[0]]['pos']
        x1, y1 = graph.nodes[edge[1]]['pos']
        xs += tuple([x0, x1, None])
        ys += tuple([y0, y1, None])

    def update(trace):
        if trace.name == "edge_trace":
            trace.x = xs
            trace.y = ys

    # do update
    fig.for_each_trace(update)

    print("update out")
    return fig


app = Dash(__name__)

if __name__ == "__main__":
    n, actions = load_logs()
    actions_count = len(actions)
    graph = load_graph(n, actions, actions_count)
    set_layout(graph, "circular_layout")
    fig = get_fig(graph)

    app.layout = html.Div([
        html.H4('Interactive color selection with simple Dash example'),
        html.P("Select color:"),
        dcc.Dropdown(
            id="dropdown",
            options=['Gold', 'MediumTurquoise', 'LightGreen'],
            value='Gold',
            clearable=False,
        ),
        dcc.Slider(0, actions_count, 1,
                   value=actions_count,
                   id='my-slider'
        ),
        dcc.Graph(id="graph", style={'width': '180vh', 'height': '90vh'}),
    ])



    @app.callback(
        Output("graph", "figure"),
        Input("my-slider", "value"))
    def display_color(value):
        new_edges = load_edges(actions, value)
        graph.clear_edges()
        graph.add_edges_from(new_edges)
        return update_fig_edges(graph, fig)


    app.run_server(debug=True)
