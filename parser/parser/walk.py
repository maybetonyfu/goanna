from collections import deque
from typing import cast
from unittest import case

import tree_sitter_haskell as haskell
from funcy import first
from tree_sitter import Language, Parser, Node

from parser.syntax import *
from state import *

haskell_language = Language(haskell.language())
parser = Parser(haskell_language)

parsing_error_query = haskell_language.query('(ERROR) @parsing_error')


def make_loc(nd: Node) -> Range:
    return (nd.start_point.row, nd.start_point.column), (nd.end_point.row, nd.end_point.column)


def find_missing(node: Node):
    for n in node.children:
        if n.is_missing:
            return n
        if find_missing(n) is not None:
            return n
    return None


def merge_locs(from_node: Node, to_node: Node) -> Range:
    return (from_node.start_point.row, from_node.start_point.column), (to_node.end_point.row, to_node.end_point.column)


def get_text(nd: Node) -> str:
    if nd.text is None:
        raise ValueError("Node does not contains text")
    return nd.text.decode('utf-8')

def get_children(nd: Node) -> list[Node]:
    return [n for n in nd.named_children if n.type != "comment"]

def match_literal(node: Node, env: ParseEnv) -> Lit:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    match node.type:
        case "integer":
            return LitInt(id=env.new_id(), loc=make_loc(node))
        case "string":
            return LitString(id=env.new_id(), loc=make_loc(node))
        case "char":
            return LitChar(id=env.new_id(), loc=make_loc(node))
        case "float":
            return LitFrac(id=env.new_id(), loc=make_loc(node))
        case _:
            raise HaskellParsingError(make_loc(node))


def match_pat(node: Node, env: ParseEnv) -> Pat:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    match node.type:
        case "qualified":
            module = get_text(node.child_by_field_name("module"))
            ident = get_text(node.child_by_field_name("id"))
            return PApp(id=env.new_id(), loc=make_loc(node), name=ident, canonical_name=None, pats=[], module=module)
        case "prefix_id":
            operator = node.named_child(0)
            return PVar(id=env.new_id(), loc=make_loc(node), name=get_text(operator), canonical_name=None)
        case "variable":
            return PVar(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name='')
        case "constructor":
            return PApp(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name='', pats=[],
                        module=None)
        case "literal":
            return match_literal(node.named_child(0), env)
        case "tuple":
            pats = [match_pat(child, env) for child in node.children_by_field_name("element")]
            return PTuple(id=env.new_id(), loc=make_loc(node), pats=pats)
        case "parens":
            return match_pat(node.child_by_field_name("pattern"), env)
        case "wildcard":
            return PWildCard(id=env.new_id(), loc=make_loc(node))
        case "apply":
            pats = deque([])
            current_node = node
            while True:
                if current_node.type == "apply":
                    pats.appendleft(match_pat(current_node.child(1), env))
                    current_node = current_node.child(0)
                else:
                    pats.appendleft(match_pat(current_node, env))
                    break
            pats = list(pats)
            return PApp(id=env.new_id(), loc=make_loc(node), name=pats[0].name, pats=pats[1:], canonical_name=None,
                        module=pats[0].module)
        case "infix":
            left_operand = match_pat(node.child_by_field_name("left_operand"), env)
            right_operand = match_pat(node.child_by_field_name("right_operand"), env)
            operator_node = node.child_by_field_name("operator")
            operator = None
            module = None
            if operator_node.type == 'variable':
                operator = get_text(operator_node)
            elif operator_node.type == 'qualified':
                operator = get_text(operator_node.child_by_field_name("id"))
                module = get_text(operator_node.child_by_field_name("module"))
            elif operator_node.type == 'constructor_operator':
                operator = get_text(operator_node)
            return PInfix(id=env.new_id(), loc=make_loc(node), pat1=left_operand, pat2=right_operand, name=operator,
                          canonical_name=None, module=module)

        case "list":
            pats = [match_pat(child, env) for child in node.children_by_field_name("element")]
            return PList(id=env.new_id(), loc=make_loc(node), pats=pats)

        case _:
            print(node.type, node.text)
            raise HaskellParsingError(make_loc(node))


def match_alt(node: Node, env: ParseEnv) -> Alt:
    pat = match_pat(node.child_by_field_name("pattern"), env)
    exp = match_exp(node.child_by_field_name("match").child_by_field_name('expression'), env)
    bind_node = node.child_by_field_name("binds")
    decl_nodes = [] if bind_node is None else bind_node.children_by_field_name("decl")
    decls = [match_decl(child, env) for child in decl_nodes]
    return Alt(id=env.new_id(), loc=make_loc(node), pat=pat, exp=exp, binds=decls)


def match_exp(node: Node, env: ParseEnv) -> Exp:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    match node.type:
        case "qualified":
            module = get_text(node.child_by_field_name("module"))
            ident = node.child_by_field_name("id")
            match ident.type:
                case 'variable':
                    return ExpVar(id=env.new_id(), loc=make_loc(node), name=get_text(ident), canonical_name=None,
                                  module=module)
                case 'constructor':
                    return ExpCon(id=env.new_id(), loc=make_loc(node), name=get_text(ident), canonical_name=None,
                                  module=module)
        case "variable":
            return ExpVar(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name=None, module=None)
        case "parens":
            return match_exp(node.child_by_field_name("expression"), env)
        case "unit":
            return ExpCon(id=env.new_id(), loc=make_loc(node), name="unit", canonical_name=None, module=None)
        case "constructor":
            return ExpCon(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name=None, module=None)

        case "prefix_id":
            return match_exp(node.named_child(0), env)

        case "operator":
            return ExpVar(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name=None, module=None)

        case "apply":
            return ExpApp(id=env.new_id(), loc=make_loc(node), exp1=match_exp(node.child(0), env),
                          exp2=match_exp(node.child(1), env))

        case "infix":
            operator_node = node.child_by_field_name("operator")
            left_operand = match_exp(node.child_by_field_name("left_operand"), env)
            right_operand = match_exp(node.child_by_field_name("right_operand"), env)
            operator = ''
            module = None
            if operator_node.type == 'constructor_operator' or operator_node.type == 'operator':
                operator = get_text(operator_node)
            elif operator_node.type == 'qualified':
                operator = get_text(operator_node.child_by_field_name("id"))
                module = get_text(operator_node.child_by_field_name("module"))
            elif operator_node.type == 'infix_id':
                back_stick_operator = operator_node.named_child(0)
                if back_stick_operator.type == 'variable' or back_stick_operator.type == 'constructor_operator':
                    operator = get_text(back_stick_operator)
                elif back_stick_operator.type == 'qualified':
                    operator = get_text(back_stick_operator.child_by_field_name("id"))
                    module = get_text(back_stick_operator.child_by_field_name("module"))
            return ExpInfixApp(id=env.new_id(), loc=make_loc(node), exp1=left_operand, name=operator, module=module,
                               canonical_name=None, exp2=right_operand)

        case "left_section":
            left_operand = node.child_by_field_name("left_operand")
            operator = node.child_by_field_name("operator")
            left = match_exp(left_operand, env)
            op = match_exp(operator, env)
            return ExpLeftSection(id=env.new_id(), loc=make_loc(node), left=left, op=op)
        case "right_section":
            operator = node.child_by_field_name("operator")
            right_operand = node.child_by_field_name("right_operand")
            right = match_exp(right_operand, env)
            op = match_exp(operator, env)
            return ExpRightSection(id=env.new_id(), loc=make_loc(node), right=right, op=op)

        case "lambda":
            pats = [match_pat(child, env) for child in node.child_by_field_name("patterns").children]
            exp = match_exp(node.child_by_field_name("expression"), env)
            return ExpLambda(id=env.new_id(), loc=make_loc(node), pats=pats, exp=exp)

        case "let_in":
            binds = [match_decl(child, env) for child in node.child_by_field_name("binds").children]
            exp = match_exp(node.child_by_field_name("expression"), env)
            return ExpLet(id=env.new_id(), loc=make_loc(node), binds=binds, exp=exp)

        case "conditional":
            cond_exp = match_exp(node.child_by_field_name("if"), env)
            then_exp = match_exp(node.child_by_field_name("then"), env)
            else_exp = match_exp(node.child_by_field_name("else"), env)
            return ExpIf(id=env.new_id(), loc=make_loc(node), cond=cond_exp, if_true=then_exp, if_false=else_exp)

        case "case":
            exp = match_exp(node.child(1), env)
            alternatives = node.child_by_field_name("alternatives")
            alts = [match_alt(child, env) for child in alternatives.children]
            return ExpCase(id=env.new_id(), loc=make_loc(node), exp=exp, alts=alts)

        case "do":
            stmts = []
            for child in node.children_by_field_name("statement"):
                match child.type:
                    case "exp":
                        qualifier = Qualifier(id=env.new_id(), loc=make_loc(node),
                                              exp=match_exp(child.named_child(0), env))
                        stmts.append(qualifier)
                    case "bind":
                        pat = match_pat(child.child_by_field_name("pattern"), env)
                        exp_node = child.child_by_field_name("expression")
                        exp = match_exp(exp_node, env)
                        gen = Generator(id=env.new_id(), loc=make_loc(node), pat=pat, exp=exp)
                        stmts.append(gen)
                    case "let":
                        decls = [match_decl(child, env) for child in
                            child.child_by_field_name('binds').children_by_field_name('decl')]
                        let = LetStmt(id=env.new_id(), loc=make_loc(node), binds=decls)
                        stmts.append(let)
            return ExpDo(id=env.new_id(), loc=make_loc(node), stmts=stmts)
        case "tuple":
            exps = [match_exp(child, env) for child in node.children_by_field_name("element")]
            return ExpTuple(id=env.new_id(), loc=make_loc(node), exps=exps)

        case "list":
            exps = [match_exp(child, env) for child in node.children_by_field_name("element")]
            return ExpList(id=env.new_id(), loc=make_loc(node), exps=exps)

        case "arithmetic_sequence":
            start = node.child_by_field_name("from")
            end = node.child_by_field_name("to")
            if start is None:
                exp = match_exp(end, env)
                return ExpEnumTo(id=env.new_id(), loc=make_loc(node), exp=exp)
            elif end is None:
                exp = match_exp(start, env)
                return ExpEnumFrom(id=env.new_id(), loc=make_loc(node), exp=exp)
            else:
                exp1 = match_exp(start, env)
                exp2 = match_exp(end, env)
                return ExpEnumFromTo(id=env.new_id(), loc=make_loc(node), exp1=exp1, exp2=exp2)

        case "literal":
            return match_literal(node.named_child(0), env)

        case _:
             raise HaskellParsingError(make_loc(node))


def match_rhs(node: Node, env: ParseEnv) -> Rhs:
    rhs_nodes = node.children_by_field_name("match")
    wheres_node = node.child_by_field_name("binds")
    wheres = [] if wheres_node is None else [match_decl(child, env) for child in
                                             wheres_node.children_by_field_name("decl")]
    patterns_node = node.child_by_field_name("patterns")
    is_pat_binding = patterns_node is None
    branches = []
    is_unguarded = rhs_nodes[0].child_by_field_name('guards') is None
    for rhs_node in rhs_nodes:
        if is_pat_binding:
            rhs_exp = match_exp(rhs_node.child_by_field_name("expression"), env)
        else:  # is function binding
            pats = [match_pat(p, env) for p in get_children(patterns_node)]
            rhs_exp = ExpLambda(id=env.new_id(), loc=merge_locs(patterns_node, rhs_node), pats=pats,
                exp=match_exp(rhs_node.child_by_field_name("expression"), env))
        if is_unguarded:
            loc = make_loc(rhs_node)
            if rhs_node.text.decode('utf8').startswith('='):
                loc = [[loc[0][0], loc[0][1] + 1], loc[1]]
            return UnguardedRhs(id=env.new_id(), loc=loc, exp=rhs_exp, wheres=wheres)
        else:
            guards_node = rhs_node.child_by_field_name("guards")
            guard_nodes = [g.named_child(0) for g in guards_node.children_by_field_name("guard")]
            guards = [match_exp(g, env) for g in guard_nodes]
            branches.append(GuardBranch(id=env.new_id(), loc=make_loc(rhs_node), exp=rhs_exp, guards=guards))

    return GuardedRhs(id=env.new_id(), loc=make_loc(node), branches=branches, wheres=wheres)


def match_type(node: Node, env: ParseEnv) -> Ty:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    match node.type:
        case "qualified":
            module = get_text(node.child_by_field_name("module"))
            ident = node.child_by_field_name("id")
            return TyCon(id=env.new_id(), loc=make_loc(node), name=get_text(ident), canonical_name=None, module=module)
        case "context":
            context = match_context(node.child_by_field_name("context"), env)
            ty = match_type(node.child_by_field_name("type"), env)
            return TyForall(id=env.new_id(), loc=make_loc(node), ty=ty, context=context)
        case "unit":
            return TyCon(id=env.new_id(), loc=make_loc(node), name="Top", canonical_name=None, module=None)
        case "name":
            return TyCon(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name=None, module=None)
        case "variable":
            return TyVar(id=env.new_id(), loc=make_loc(node), name=get_text(node), canonical_name=None)
        case "apply":
            node1 = node.child_by_field_name("constructor")
            node2 = node.child_by_field_name("argument")
            ty1 = match_type(node1, env)
            ty2 = match_type(node2, env)
            return TyApp(id=env.new_id(), loc=make_loc(node), ty1=ty1, ty2=ty2)
        case "parens":
            return match_type(node.child_by_field_name("type"), env)
        case "function":
            node1 = node.child_by_field_name("parameter")
            node2 = node.child_by_field_name("result")
            ty1 = match_type(node1, env)
            ty2 = match_type(node2, env)
            return TyFun(id=env.new_id(), loc=make_loc(node), ty1=ty1, ty2=ty2)

        case "tuple":
            tys = [match_type(child, env) for child in node.children_by_field_name("element")]
            return TyTuple(id=env.new_id(), loc=make_loc(node), tys=tys)

        case "list":
            ty = match_type(node.child_by_field_name("element"), env)
            return TyList(id=env.new_id(), loc=make_loc(node), ty=ty)
        case _:
            raise HaskellParsingError(make_loc(node))


def match_context(node: Node, env: ParseEnv) -> Context:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    tys = []
    match node.type:
        case "parens":
            tys = [match_type(node.child(1), env)]
        case "tuple":
            # Multiple contex
            tys = [match_type(child, env) for child in get_children(node)]
        case "apply":
            # Single context
            tys = [match_type(node, env)]
        case _:
            raise HaskellParsingError(make_loc(node))

    return Context(id=env.new_id(), loc=make_loc(node), assertions=tys)


def match_decl_head(node: Node, env: ParseEnv) -> DeclHead:
    name = get_text(node.child_by_field_name("name"))
    patterns_node = node.child_by_field_name("patterns")
    patterns = [] if patterns_node is None else patterns_node.children_by_field_name('bind')
    ty_vars = [match_type(t, env) for t in patterns]
    return DeclHead(id=env.new_id(), loc=make_loc(node), name=name, ty_vars=ty_vars, canonical_name=None)


def match_decl(node: Node, env: ParseEnv) -> Decl:
    if node.is_missing:
        raise HaskellParsingError(make_loc(node))
    match node.type:

        case "signature":
            names = []
            if name_binds := node.child_by_field_name("names"):
                name_nodes = name_binds.children_by_field_name("name")
            else:
                name_nodes = [node.child_by_field_name("name")]
            for child in name_nodes:
                if child.type == "prefix_id":
                    names.append(get_text(child.named_child(0)))
                else:
                    names.append(get_text(child))

            return TypeSig(id=env.new_id(), loc=make_loc(node), names=names,
                ty=match_type(node.child_by_field_name("type"), env), canonical_names=[])

        case "data_type":
            data_head = match_decl_head(node, env)
            constructor_node = node.child_by_field_name("constructors")
            constructors = []
            if constructor_node:
                constructors = constructor_node.children_by_field_name("constructor")
            data_cons = []
            for c in constructors:
                data_con_node = c.child_by_field_name('constructor')
                name_node = data_con_node.child_by_field_name('name')
                name = get_text(name_node)
                field_nodes = [match_type(fn, env) for fn in data_con_node.children_by_field_name('field')]
                data_cons.append(
                    DataCon(id=env.new_id(), loc=make_loc(c), name=name, tys=field_nodes, canonical_name=None))
            return DataDecl(id=env.new_id(), loc=make_loc(node), d_head=data_head, constructors=data_cons, deriving=[])

        case "class":
            context_node = node.child_by_field_name("context")
            context = None
            if context_node:
                context = match_context(node.child_by_field_name("context").child_by_field_name('context'), env)
            d_head = match_decl_head(node, env)
            decl_node = node.child_by_field_name("declarations")
            decls = []
            if decl_node is not None:
                decls = [match_decl(child, env) for child in get_children(decl_node)]
            return ClassDecl(id=env.new_id(), loc=make_loc(node), context=context, d_head=d_head, decls=decls)

        case "instance":
            context_node = node.child_by_field_name("context")
            context = None
            if context_node:
                context = match_context(context_node.child_by_field_name('context'), env)
            name_node = node.child_by_field_name("name")
            if name_node.type == "qualified":
                module = get_text(name_node.child_by_field_name("module"))
                name = get_text(name_node.child_by_field_name("id"))
            else:
                module = None
                name = get_text(name_node)
            patterns_node = node.child_by_field_name("patterns")
            patterns = [] if patterns_node is None else get_children(patterns_node)
            tys = [match_type(t, env) for t in patterns]
            body_declaration_node = node.child_by_field_name("declarations")
            decls = []
            if body_declaration_node:
                decls = [match_decl(child, env) for child in get_children(body_declaration_node)]
            return InstDecl(id=env.new_id(), loc=make_loc(node), context=context, name=name, tys=tys, body=decls,
                            canonical_name=None, module=module)

        case "function" | "bind":
            variable_node = node.named_child(0)
            pat = match_pat(variable_node, env)
            rhs = match_rhs(node, env)
            return PatBind(id=env.new_id(), loc=make_loc(node), pat=pat, rhs=rhs)

        case _:
            raise HaskellParsingError(make_loc(node))



def make_ast(node: Node, env: ParseEnv, module_name_alt: str | None = None) -> Module | None:
    parsing_errors = parsing_error_query.captures(node)
    if parsing_errors:
        raise HaskellParsingError(make_loc(parsing_errors['parsing_error'][0]))
    missing_node = find_missing(node)
    if missing_node is not None:
        raise HaskellParsingError(make_loc(missing_node))
    decl_node = node.child_by_field_name("declarations")
    decls = []
    if decl_node:
        decls = [match_decl(d, env) for d in get_children(decl_node)]

    import_node = node.child_by_field_name('imports')
    imports = []
    if import_node:
        imports = [make_import(i) for i in get_children(import_node)]

    module = first(child.child_by_field_name('module') for child in node.children if child.type == "header")
    module_ids = [get_text(m) for m in get_children(module)] if module else []
    module_name = '.'.join(module_ids)
    module_name = module_name_alt if module_name == '' and module_name_alt else module_name

    return Module(id=env.new_id(), loc=make_loc(node), name=cast(str, module_name), imports=imports, decls=decls)


def make_import(node: Node) -> str:
    module = node.child_by_field_name('module')
    module_ids = [get_text(m) for m in get_children(module)] if module else []
    module_name = '.'.join(module_ids)
    return module_name


def parse_haskell(code: str) -> Node:
    return parser.parse(bytes(code, 'utf-8')).root_node


if __name__ == "__main__":
    tree = parse_haskell("""
x :: [Char]
x = '4'
""")
    print(tree)
    # query = haskell_language.query('(ERROR) @parsing_error')
    # captures = query.captures(tree)
    # print(captures)
    # missings = find_missing(tree)
    # print('missing: ', missings)
    ast = make_ast(tree, ParseEnv(), 'Test')
    print(ast)