from parser.traverse import Traverse
from parser.syntax import *
from state import Range

def update_node_table(data: dict[int, Range], ast: Pretty, parent: Pretty, *_) -> dict[int, Range]:
    if not ast:
        print(ast, parent)
        raise Exception("Error updating node table")
    data[ast.id] = ast.loc
    return data



def gather_node_table(asts: list[Pretty]) -> dict[int, Range]:
    traverser = Traverse(
        init_data={},
        update_func=update_node_table)
    traverser.traverse_all(asts)
    return traverser.value

