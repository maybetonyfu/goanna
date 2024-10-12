from itertools import chain

from parser.traverse import Traverse
from parser.syntax import *
from state import *
from scope import canonical_names_from_pat


class ArgumentGatherer(BaseModel):
    current_decl: str
    arguments: dict[str, set[str]]


def update_arguments(data: ArgumentGatherer, ast: Pretty, *_) -> ArgumentGatherer:
    match ast:
        case PatBind(pat=PVar(canonical_name=canonical_name)):
            data.current_decl = canonical_name
            return data

        case Alt(pat=pat):
            names = canonical_names_from_pat(pat)
            for name in names:
                data.arguments.setdefault(data.current_decl, set()).add(name)
            return data

        case ExpLambda(pats=pats):
            for pat in pats:
                names = canonical_names_from_pat(pat)
                for name in names:
                    data.arguments.setdefault(data.current_decl, set()).add(name)
            return data

        case _:
            return data


def gather_arguments(asts: list[Pretty], closures: Closures) -> Arguments:
    # print(closures)
    data = ArgumentGatherer(current_decl='', arguments={})
    traverser = Traverse(init_data=data, update_func=update_arguments)
    traverser.traverse_all(asts)
    data = traverser.value
    arguments: Arguments = data.arguments
    # print("Old Argument: ", arguments)
    _arguments: Arguments = {}

    for child, parents in closures.items():
        _arguments[child] = [*list(chain(*[arguments.get(p, []) for p in parents])), *arguments.get(child, [])]
    # for decl, args in arguments.items():
    #     _arguments[decl] = list(chain(
    #         *[arguments[parent] for parent in closures[decl]], args
    #     ))
    # print("New Argument: ", _arguments)
    return _arguments
