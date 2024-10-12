import itertools
from functools import partial
from typing import cast
from state import *

from encode import encode
from state import State
from parser.syntax import *
from parser.traverse import Traverse

declaration_env: set[str] = set()
parameter_env: set[str] = set()
type_env: set[str] = set()
class_env: set[str] = set()
synonym_env: set[str] = set()



GLOBAL = EffectiveRange(ranges=None, is_global=True, excludes=[])


def names_from_pat(pat: Pat) -> list[tuple[str, int]]:
    match pat:
        case PVar(name=name, id=id):
            return [(name, id)]
        case PApp(pats=pats) | PList(pats=pats) | PTuple(pats=pats):
            return list(itertools.chain(*[names_from_pat(p) for p in pats]))
        case PWildCard() | LitFrac() | LitString() | LitInt() | LitChar():
            return []
        case PInfix(pat1=pat1, pat2=pat2):
            return names_from_pat(pat1) + names_from_pat(pat2)

def canonical_names_from_pat(pat: Pat) -> list[str]:
    match pat:
        case PVar(canonical_name=canonical_name):
            if canonical_name is None:
                raise ValueError("No canonical name found")
            return [canonical_name]
        case PApp(pats=pats) | PList(pats=pats) | PTuple(pats=pats):
            return list(itertools.chain(*[canonical_names_from_pat(p) for p in pats]))
        case PWildCard() | LitFrac() | LitString() | LitInt() | LitChar():
            return []
        case PInfix(pat1=pat1, pat2=pat2):
            return canonical_names_from_pat(pat1) + canonical_names_from_pat(pat2)

def get_effective_range(node_id: int, parent: Pretty) -> EffectiveRange:
    match parent:
        case Module():
            return GLOBAL
        case ClassDecl():
            return GLOBAL
        case InstDecl():
            return GLOBAL
        case GuardedRhs(wheres=wheres):
            excludes = []
            for where in wheres:
                if node_id == where.id:
                    break
                excludes.append(where.loc)
            return EffectiveRange(ranges=parent.loc, excludes=excludes)
        case UnguardedRhs(wheres=wheres):
            excludes = []
            for where in wheres:
                if node_id == where.id:
                    break
                excludes.append(where.loc)
            return EffectiveRange(ranges=parent.loc, excludes=excludes)
        case Alt(binds=binds):
            excludes = []
            for where in binds:
                if node_id == where.id:
                    break
                excludes.append(where.loc)
            return EffectiveRange(ranges=parent.loc, excludes=excludes)
        case ExpLet(binds=binds):
            excludes = []
            for where in binds:
                if node_id == where.id:
                    break
                excludes.append(where.loc)
            return EffectiveRange(ranges=parent.loc, excludes=excludes)
        case _:
            raise ValueError("Cannot get effective range")


def update_vendors(module_mapping: dict[str, str], module_name: str, data: list[Vendor], ast: Pretty, parent: Pretty, *_) -> list[Vendor]:
    match ast:
        case TypeSig(names=names):
            effective_range = get_effective_range(ast.id, parent)
            # if effective_range.ranges is None:
            #     raise ValueError("Effective range is None")
            for name in names:
                if effective_range.is_global:
                    canonical_name = f'{module_mapping[module_name]}_{encode(name)}'
                else:
                    line = effective_range.ranges[0][0]
                    column = effective_range.ranges[0][1]
                    canonical_name = f'{module_mapping[module_name]}_{encode(name)}_{line}_{column}'
                data.append(Vendor(
                    node_id=ast.id,
                    name=name,
                    type='term',
                    module=module_name,
                    is_declaration=True,
                    canonical_name=canonical_name,
                    effective_range=effective_range,
                ))

        case PatBind(pat=PVar(name=name, id=id)):
            if isinstance(parent, LetStmt):
                return data
            effective_range = get_effective_range(ast.id, parent)
            if effective_range.is_global:
                canonical_name = f'{module_mapping[module_name]}_{encode(name)}'
            else:
                if effective_range.ranges is None:
                    raise ValueError("Effective range is None")
                line = effective_range.ranges[0][0]
                column = effective_range.ranges[0][1]
                canonical_name = f'{module_mapping[module_name]}_{encode(name)}_{line}_{column}'
            data.append(Vendor(
                node_id=id,
                name=name,
                module=module_name,
                is_declaration=True,
                type='term',
                canonical_name=canonical_name,
                effective_range=effective_range,
            ))

        case DataCon(name=name, id=id):
            canonical_name = f'{module_mapping[module_name]}_{name}'
            data.append(Vendor(
                node_id=id,
                name=name,
                module=module_name,
                type='term',
                is_declaration=True,
                canonical_name=canonical_name,
                effective_range=GLOBAL,
            ))

        case DataDecl(d_head=d_head):
            canonical_name = f'{module_mapping[module_name]}_{d_head.name}'
            data.append(Vendor(
                node_id=d_head.id,
                name=d_head.name,
                type='type',
                module=module_name,
                canonical_name=canonical_name,
                effective_range=GLOBAL,
            ))

        case ClassDecl(d_head=d_head):
            canonical_name = f'{module_mapping[module_name]}_{d_head.name}'
            data.append(Vendor(
                node_id=d_head.id,
                name=d_head.name,
                type='type',
                module=module_name,
                canonical_name=canonical_name,
                effective_range=GLOBAL,
            ))

        case ExpDo(stmts=stmts):
            end = ast.loc[1]
            for stmt in stmts:
                if isinstance(stmt, LetStmt):
                    for bind in stmt.binds:
                        if isinstance(bind, PatBind):
                            start = bind.loc[1]
                            line = start[0]
                            column = start[1]
                            effective_range = EffectiveRange(ranges=(start, end), excludes=[])
                            for name, id in names_from_pat(bind.pat):
                                canonical_name = f'{module_mapping[module_name]}_{encode(name)}_{line}_{column}'
                                data.append(Vendor(
                                    node_id=id,
                                    name=name,
                                    type='term',
                                    is_declaration=True,
                                    module=module_name,
                                    canonical_name=canonical_name,
                                    effective_range=effective_range,
                                ))

        # Parameters
        case ExpLambda(pats=pats):
            line = ast.loc[0][0]
            column = ast.loc[0][1]
            for pat in pats:
                names = names_from_pat(pat)
                effective_range = EffectiveRange(ranges=ast.loc, excludes=[])
                for name, id in names:
                    canonical_name = f'{module_mapping[module_name]}_{encode(name)}_{line}_{column}'
                    data.append(Vendor(
                        node_id=id,
                        name=name,
                        type='term',
                        module=module_name,
                        canonical_name=canonical_name,
                        effective_range=effective_range,
                    ))

        case Alt(pat=pat):
            effective_range = EffectiveRange(ranges=ast.loc, excludes=[])
            line = ast.loc[0][0]
            column = ast.loc[0][1]
            for name, id in names_from_pat(pat):
                canonical_name = f'{module_mapping[module_name]}_{encode(name)}_{line}_{column}'
                data.append(Vendor(
                    node_id=id,
                    name=name,
                    type='term',
                    module=module_name,
                    canonical_name=canonical_name,
                    effective_range=effective_range,
                ))
    return data


def get_vendors(asts: list[Pretty], state: State) -> list[Vendor]:
    vendors = []
    for ast in asts:
        module_name = cast(Module, ast).name
        traverser = Traverse(init_data=[], update_func=partial(update_vendors, state.module_mapping, module_name))
        traverser.traverse(ast)
        _vendors: list[Vendor] = traverser.value
        vendors.extend(_vendors)
    return vendors


def update_buyers(module_name: str, data: list[Buyer], ast: Pretty, *_) -> list[Buyer]:
    match ast:
        case ExpVar(name=name, module=module) | ExpCon(name=name, module=module) | \
             PApp(name=name, module=module) | PInfix(name=name, module=module) | \
             ExpInfixApp(name=name, module=module):
            data.append(Buyer(
                node_id=ast.id,
                name=name,
                type="term",
                module=module,
                usage_module=module_name,
                usage_loc=ast.loc,
                canonical_name=None
            ))

        case TyCon(name=name, module=module):
            data.append(Buyer(
                node_id=ast.id,
                name=name,
                type="type",
                module=module,
                usage_module=module_name,
                usage_loc=ast.loc,
                canonical_name=None
            ))

        case InstDecl(name=name, module=module):
            data.append(Buyer(
                node_id=ast.id,
                name = name,
                type="type",
                module=module,
                usage_module=module_name,
                usage_loc=ast.loc,
                canonical_name=None
            ))

    return data


def get_buyers(asts: list[Pretty]) -> list[Buyer]:
    buyers = []
    for ast in asts:
        module_name = cast(Module, ast).name
        traverser = Traverse(init_data=[], update_func=partial(update_buyers, module_name))
        traverser.traverse(ast)
        _buyers: list[Buyer] = traverser.value
        buyers.extend(_buyers)
    return buyers



def in_range(buyer: Buyer, vendor: Vendor, import_map: dict[str, list[str]]) -> bool:
    if vendor.name != buyer.name:
        return False
    if buyer.module is not None and vendor.module != buyer.module:
        return False
    if vendor.type != buyer.type:
        return False

    if vendor.effective_range.is_global:
        allowed_imports = [buyer.usage_module, *import_map.get(buyer.usage_module, [])]
        return vendor.module in allowed_imports
    else:
        return vendor.module == buyer.usage_module and \
            within(buyer.usage_loc, vendor.effective_range.ranges) and \
            not any(within(buyer.usage_loc, r) for r in vendor.effective_range.excludes)


def allocate_buyers(vendors: list[Vendor], buyers: list[Buyer], import_map: dict[str, list[str]]) -> tuple[
                list[Buyer], list[Buyer]]:
    new_buyers = []
    import_errors = []
    for buyer in buyers:
        vs = [v for v in vendors if in_range(buyer, v, import_map)]
        if len(vs) == 0:
            match buyer.name:
                case 'undefined':
                    buyer.canonical_name = 'builtin_bottom'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)
                case 'unit':
                    buyer.canonical_name = 'builtin_unit'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)

                case 'Top':
                    buyer.canonical_name = 'builtin_Top'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)

                case ':':
                    buyer.canonical_name = 'builtin_cons'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)
                case 'Int':
                    buyer.canonical_name = 'builtin_Int'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)
                case 'Char':
                    buyer.canonical_name = 'builtin_Char'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)
                case 'Float':
                    buyer.canonical_name = 'builtin_Float'
                    buyer.module = 'builtin'
                    new_buyers.append(buyer)
                case _:
                    import_errors.append(buyer)
            continue

        smallest = vs[0]
        for v in vs[1:]:
            if smallest.effective_range.is_global:
                smallest = v
            else:
                if after(v.effective_range.ranges[0], smallest.effective_range.ranges[0]):
                    smallest = v
        buyer.canonical_name = smallest.canonical_name
        buyer.module = smallest.module
        new_buyers.append(buyer)
    return new_buyers, import_errors
