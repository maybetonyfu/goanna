from typing import cast

from arguments import gather_arguments
from closure import gather_closures
from constraint import get_all_constraints, GlobalState
from node_depth import gather_label
from node_graph import gather_node_graph
from node_table import gather_node_table
from parser.walk import parse_haskell, make_ast
from rename import rename
from scope import get_vendors, get_buyers, allocate_buyers
from state import HaskellParsingError, ParseEnv
from state import State, Closures
from typeclass import gather_classes
from typevar import gather_type_vars


def parse_modules(files: list[tuple[str, str]]) -> State:
    asts = []
    state = State()
    import_map: dict[str, list[str]] = {}
    parse_env = ParseEnv()

    for i, (module_name, file_content) in enumerate(files):
        tree = parse_haskell(file_content)
        try:
            module = make_ast(tree, parse_env, module_name)
            asts.append(module)
            import_map[module.name] = module.imports
            if module.name != 'Prelude' and "Prelude" not in module.imports:
                module.imports.append('Prelude')

            if module.name == 'Prelude':
                state.module_mapping[module.name] = 'p'
            else:
                state.module_mapping[module.name] = 'm' + str(i)

            state.text_lines[module.name] = file_content.split('\n')

        except HaskellParsingError as e:
            state.parsing_errors.append(e.loc)
            break

    if state.parsing_errors:
        return state
    vendors = get_vendors(asts, state)
    state.declarations = list({v.canonical_name for v in vendors if v.type == 'term' and v.is_declaration})
    buyers = get_buyers(asts)
    buyers, import_errors = allocate_buyers(vendors, buyers, import_map)
    state.import_errors = import_errors
    if import_errors:
        return state
    for ast in asts:
        rename(ast, vendors, buyers)

    state.asts = asts
    node_table = gather_node_table(state.asts)
    state.node_table = node_table
    return state


def static_analysis(state: State) -> State:
    asts = state.asts
    state.node_depth = gather_label(asts)
    state.max_depth = max(state.node_depth.values())
    state.node_graph = gather_node_graph(asts)
    get_all_constraints(asts, cast(GlobalState, state))
    closures: Closures = gather_closures(asts)
    state.arguments = gather_arguments(asts, closures)
    state.classes = gather_classes(asts)
    state.type_vars = gather_type_vars(asts, state.classes)
    return state

def run_modules(files: list[tuple[str, str]]) -> State:
    state = parse_modules(files)
    if state.parsing_errors or state.import_errors:
        return state

    state = static_analysis(state)
    return state
