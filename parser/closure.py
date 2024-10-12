from parser.syntax import *
from parser.traverse import Traverse
from state import *


class ClosureGatherer(BaseModel):
    decl_stack: list[str]
    closures: dict[str, list[str]]


def update_closure(data: ClosureGatherer, ast: Pretty, *_) -> ClosureGatherer:
    match ast:
        case PatBind(pat=PVar(canonical_name=canonical_name)):
            # print("Current: ", canonical_name)
            # print("Parents: ", data.decl_stack)
            data.closures[canonical_name] = data.decl_stack.copy()
            data.decl_stack.append(canonical_name)
            # print("Parents Updated: ", data.decl_stack)
    return data


def cleanup_closure(data: ClosureGatherer, ast: Pretty, *_) -> ClosureGatherer:
    match ast:
        case PatBind(pat=PVar()):
            data.decl_stack.pop()
    return data


def gather_closures(asts: list[Pretty]) -> Closures:
    data = ClosureGatherer(decl_stack=[], closures={})
    traverser = Traverse(init_data=data, update_func=update_closure)
    traverser.set_cleanup(cleanup_closure)
    traverser.traverse_all(asts)
    data: ClosureGatherer = traverser.value
    return data.closures
