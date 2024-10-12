from parser.traverse import Traverse
from parser.syntax import *
type NodeDepth = dict[int, int]

def update_label(data: NodeDepth, ast: Pretty, parent: Pretty) -> NodeDepth:
    if parent is None:
        data[ast.id] = 0
    else:
        data[ast.id] = data[parent.id] + 1
    return data


def gather_label(asts: list[Pretty]) -> NodeDepth:
    traverser = Traverse(init_data={}, update_func=update_label)
    traverser.traverse_all(asts)
    return traverser.value
