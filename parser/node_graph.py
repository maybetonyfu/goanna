from parser.traverse import Traverse
from parser.syntax import *


def enter_node(data: list[tuple[int, int]], ast: Pretty, parent: Pretty) -> list[tuple[int, int]]:
    if parent is not None:
        data.append((parent.id, ast.id))

    return data



def gather_node_graph(asts: list[Pretty]) -> list[tuple[int, int]]:
    traverser = Traverse(
        init_data=[],
        update_func=enter_node)
    traverser.traverse_all(asts)
    return traverser.value
