from functools import partial

from parser.syntax import *
from parser.traverse import Traverse
from state import *


def update_rename(vendors: list[Vendor], buyers: list[Buyer], _unused, ast: Pretty, *_) -> None:
    match ast:
        # Vendors
        case PVar() | DataCon() | DeclHead():
            for vendor in vendors:
                if vendor.node_id == ast.id:
                    ast.canonical_name = vendor.canonical_name

        case TypeSig(names=names):
            _vendors = {v.name: v.canonical_name for v in vendors if v.node_id == ast.id}
            _names: list[str] = [_vendors[n] for n in names]
            ast.canonical_names = _names

        # Buyers
        case ExpVar() | ExpCon() | PApp() | TyCon() | InstDecl() | PInfix() | ExpInfixApp():
            for buyer in buyers:
                if buyer.node_id == ast.id:
                    ast.canonical_name = buyer.canonical_name
                    ast.module = buyer.module


def rename(ast: Pretty, vendors: list[Vendor], buyers: list[Buyer]):
    traverser = Traverse(init_data=None, update_func=partial(update_rename, vendors, buyers))
    _ast = traverser.traverse(ast)
    return _ast

