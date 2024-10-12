from typing import cast

import networkx as nx

from parser.traverse import Traverse

from parser.syntax import *


def update_class(data: nx.DiGraph, ast: Pretty, *_) -> nx.DiGraph:
    match ast:
        case ClassDecl(context=context, d_head=DeclHead(canonical_name=class_name)):
            if context is not None:
                for ty in context.assertions:
                    while isinstance(ty, TyApp):
                        ty = ty.ty1
                    ty = cast(TyCon, ty)
                    super_class_name = ty.canonical_name
                    data.add_edge(class_name, super_class_name)
            else:
                data.add_node(class_name)

    return data


def gather_classes(asts: list[Pretty]) -> dict[str, set[str]]:
    traverser = Traverse(init_data=nx.DiGraph(), update_func=update_class)
    traverser.traverse_all(asts)
    data: nx.DiGraph = traverser.value
    super_classes: dict[str, set[str]] = {cls: set(nx.descendants(data, cls)) for cls in data.nodes}
    return super_classes
