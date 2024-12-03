from typing import cast
from logic import *
from parser.syntax import *
from state import *


class GlobalState:
    def get_declarations(self) -> set[str]:
        raise NotImplementedError

    def add_rule(self, rule: Rule) -> None:
        raise NotImplementedError

    def node_level(self, node_id: int) -> int:
        raise NotImplementedError

    def max_level(self) -> int:
        raise NotImplementedError

    def is_parent_of(self, parent: str, child: str) -> bool:
        raise NotImplementedError

    def add_class_var(self, head_name: str, class_var:str):
        raise NotImplementedError

class ConstraintGenState:
    def __init__(self, global_state: GlobalState):
        self.fresh_counter = 0
        self.module = ''
        self.global_state = global_state

    def set_module_name(self, module: str):
        self.module = module

    @property
    def declarations(self):
        return self.global_state.get_declarations()

    def add_rule(self, rule: LTerm, head: RuleHead, node_id: int):
        self.global_state.add_rule(Rule(head=head, body=rule, axiom=False, node_id=node_id))

    def add_rules(self, rules: list[LTerm], head: RuleHead, node_id: int):
        for rule in rules:
            self.add_rule(rule, head, node_id)

    def add_axiom(self, rule: LTerm, head: RuleHead):
        self.global_state.add_rule(Rule(head=head, body=rule, axiom=True, node_id=None))

    def fresh(self) -> LVar:
        self.fresh_counter += 1
        return LVar(value=f'_f{self.fresh_counter}')

    def head_of_typing_rule(self, name: str) -> RuleHead:
        return RuleHead(type='type', name=name, id=0, module=self.module )

    def head_of_instance_rule(self, name: str, instance_id: int) -> RuleHead:
        return RuleHead(type='instance', name=name, id=instance_id, module=self.module)

    def type_of(self, name: str, var: LVar, head: RuleHead) -> list[LStruct]:
        collector = self.fresh()
        self.global_state.add_class_var(head.name, collector.value)
        if self.global_state.is_parent_of(head.name, name):
            v = self.fresh()
            rule1 =LStruct(functor=name, args=[var, Call_, wildcard, v, wildcard, collector])
            rule2 = once(LStruct(functor='append', args=[ZetaVar, wildcard, v]))
            return [rule1, rule2]
        else:
            rule = LStruct(functor=name, args=[var, Call_, wildcard, wildcard, wildcard, collector])
            return [rule]

def pair(*terms: LTerm) -> LTerm:
    match len(terms):
        case 0:
            raise ValueError("adt needs at least one argument")
        case 1:
            return terms[0]
        case _:
            return LStruct(functor='pair', args=[pair(*terms[:-1]), terms[-1]])


def list_of(elem: LTerm) -> LTerm:
    return pair(LAtom(value='list'), elem)


def fun_of(*terms: LTerm) -> LTerm:
    match len(terms):
        case 0:
            raise ValueError("fun_of needs at least one argument")
        case 1:
            return terms[0]
        case _:
            return pair(pair(LAtom(value='function'), terms[0]), fun_of(*terms[1:]))


# either a b -> ((either a)  b)
# maybe a -> (maybe a)
# [a] -> ([] a)
# (a, b) -> ((tuple a) b)
# (a, b, c) -> (((tuple a) b) c)
# a -> b -> ((function a) b)
# a -> b -> c => ((function a) ((function b) c))

def tuple_of(*terms: LTerm) -> LTerm:
    match len(terms):
        case 0:
            raise ValueError("tuple_of needs at least one argument")
        case 1:
            return pair(LAtom(value='tuple'), terms[0])
        case _:
            last_item = terms[-1]
            init_items = terms[:-1]
            return pair(tuple_of(*init_items), last_item)

#
# def type_of(name: str, var: LVar, captures: LVar, arguments: LVar) -> LStruct:
#     return LStruct(functor=name, args=[var, Call_, captures, arguments, wildcard, Classes])


def has_class(v: LVar, class_name: str) -> LTerm:
    rule_body = LStruct(functor='member',
                        args=[LStruct(functor='with', args=[LAtom(value=class_name), v]),
                              LVar(value='_Classes')])
    return once(rule_body)


def node_var(node: Pretty) -> LVar:
    return LVar(value='_' + str(node.id))


def type_var(ty_var: TyVar, decl_head: str) -> LVar:
    return LVar(value=f'_{decl_head}_{ty_var.name}')


def get_all_constraints(asts: list[Pretty], global_state: GlobalState) -> None:
    state = ConstraintGenState(global_state)
    for ast in asts:
        generate_constraint(ast, head=None, state=state)


def generate_constraint(ast: Pretty, head: RuleHead | None, state: ConstraintGenState):
    match ast:
        case Module(decls=decls, name=name):
            state.set_module_name(name)
            for decl in decls:
                generate_constraint(decl, None, state)

        case ClassDecl(d_head=d_head, decls=decls):
            d_head: DeclHead
            if len(d_head.ty_vars) != 1:
                raise NotImplementedError("Multi-parameter type class")
            class_name = d_head.canonical_name

            for decl in decls:
                decl: TypeSig
                names = decl.canonical_names
                ty = decl.ty
                for name in names:
                    head = state.head_of_typing_rule(name)
                    class_var = type_var(d_head.ty_vars[0], name)
                    state.add_axiom(unify(T, node_var(ty)), head)
                    state.add_axiom(has_class(class_var, class_name), head)
                    generate_constraint(ty, head, state)

        case InstDecl(context=context, canonical_name=class_name, tys=tys):
            context: Context | None
            head = state.head_of_instance_rule(class_name, ast.id)
            instance_type: Ty = tys[0]
            state.add_axiom(unify('T', node_var(instance_type)), head)
            generate_constraint(instance_type, head, state)
            if context is not None:
                for tyApp in context.assertions:
                    tyApp: TyApp
                    class_name = cast(TyCon, tyApp.ty1).canonical_name
                    instance_var = type_var(tyApp.ty2, head.name)
                    state.add_axiom(LStruct(functor=class_name, args=[instance_var]), head)

        case DataDecl(d_head=d_head, constructors=constructors, deriving=deriving):
            type_name = d_head.canonical_name
            type_vars: list[TyVar] = d_head.ty_vars
            for constructor in constructors:
                constructor: DataCon
                head = state.head_of_typing_rule(constructor.canonical_name)
                data_type = pair(LAtom(value=type_name), *[type_var(v, head.name) for v in type_vars])
                state.add_axiom(unify(T, fun_of(*[node_var(ty) for ty in constructor.tys], data_type)), head)
                for ty in constructor.tys:
                    generate_constraint(ty, head, state)

            for d in deriving:
                d = cast(TyCon, d)
                class_name = d.canonical_name
                ihead = state.head_of_instance_rule(class_name, d.id)
                data_type = pair(LAtom(value=type_name), *[LVar(value=f'_{v.name}') for v in type_vars])
                state.add_axiom(unify('T', data_type), ihead)

        case PatBind(pat=PVar(canonical_name=canonical_name), rhs=rhs):
            head = state.head_of_typing_rule(canonical_name)
            state.add_axiom(unify(T, node_var(rhs)), head)  # lhs = rhs
            generate_constraint(rhs, head, state)

        case UnguardedRhs(wheres=wheres, exp=exp):
            state.add_rule(unify(node_var(ast), node_var(exp)), head, ast.id)
            generate_constraint(exp, head, state)
            for where_ in wheres:
                generate_constraint(where_, head, state)

        case GuardedRhs(branches=branches, wheres=wheres):
            for branch in branches:
                state.add_rule(unify(node_var(ast), node_var(branch)), head, ast.id)
                generate_constraint(branch, head, state)
            for where_ in wheres:
                generate_constraint(where_, head, state)

        case GuardBranch(guards=guards, exp=exp):
            for guard in guards:
                state.add_axiom(unify(node_var(guard), 'p_Bool'), head)  # guard eval to bool
                generate_constraint(guard, head, state)
            state.add_axiom(unify(node_var(ast), node_var(exp)),
                            head)  # x | guard = exp; | guard2 = exp2 => exp1 = exp2
            generate_constraint(exp, head, state)

        case PVar(canonical_name=canonical_name):
            state.add_axiom(unify(node_var(ast), LVar(value=f'_{canonical_name}')), head)

        case PWildCard():
            pass

        case PList(pats=elems) | ExpList(exps=elems):
            fresh = state.fresh()
            for elem in elems:
                generate_constraint(elem, head, state)
            state.add_rule(unify(node_var(ast), list_of(fresh)), head, ast.id)
            state.add_rule(unify_all([node_var(elem) for elem in elems] + [fresh]), head, ast.id)

        case PInfix(pat1=pat1, pat2=pat2, canonical_name=canonical_name):
            fun_var = state.fresh()
            fun = fun_of(node_var(pat1), node_var(pat2), node_var(ast))
            state.add_rule(unify(fun, fun_var), head, ast.id)
            # state.add_rules(type_of(canonical_name, fun_var, wildcard, wildcard), head, ast.id)
            state.add_rules(state.type_of(canonical_name, fun_var, head), head, ast.id)
            generate_constraint(pat1, head, state)
            generate_constraint(pat2, head, state)


        case PApp(canonical_name=canonical_name, pats=pats):
            fun = fun_of(*[node_var(pat) for pat in pats], node_var(ast))
            v = state.fresh()
            state.add_axiom(unify(fun, v), head)
            for pat in pats:
                generate_constraint(pat, head, state)
            # state.add_rule(type_of(canonical_name, v, wildcard, wildcard), head, ast.id)
            state.add_rules(state.type_of(canonical_name, v, head), head, ast.id)

        case PTuple(pats=pats):
            state.add_axiom(unify(node_var(ast), tuple_of(*[node_var(pat) for pat in pats])), head)
            for pat in pats:
                generate_constraint(pat, head, state)

        case TypeSig(ty=ty, canonical_names=canonical_names):
            for name in canonical_names:
                head = state.head_of_typing_rule(name)
                state.add_axiom(unify(T, node_var(ty)), head)
                generate_constraint(ty, head, state)

        case TyVar(axiom=axiom):
            if axiom:
                state.add_axiom(unify(node_var(ast), type_var(cast(TyVar, ast), head.name)), head)
            else:
                state.add_rule(unify(node_var(ast), type_var(cast(TyVar, ast), head.name)), head, ast.id)


        case TyCon(canonical_name=canonical_name, axiom=axiom):
            if axiom:
                state.add_axiom(unify(node_var(ast), LAtom(value=canonical_name)), head)
            else:
                state.add_rule(unify(node_var(ast), LAtom(value=canonical_name)), head, ast.id)


        case TyForall(context=context, ty=ty, axiom=axiom):
            if context is not None:
                for tyApp in context.assertions:
                    tyApp: TyApp
                    ty1: TyCon = tyApp.ty1
                    class_name = ty1.canonical_name
                    instance_var = type_var(tyApp.ty2, head.name)
                    state.add_rule(has_class(instance_var, class_name), head, tyApp.id)
            if axiom:
                state.add_axiom(unify(node_var(ast), node_var(ty)), head)
            else:
                state.add_rule(unify(node_var(ast), node_var(ty)), head, ast.id)

            generate_constraint(ty, head, state)

        case TyApp(ty1=ty1, ty2=ty2, axiom=axiom):
            generate_constraint(ty1, head, state)
            generate_constraint(ty2, head, state)
            if axiom:
                state.add_axiom(unify(node_var(ast), pair(node_var(ty1), node_var(ty2))), head)
            else:
                state.add_rule(unify(node_var(ast), pair(node_var(ty1), node_var(ty2))), head, ast.id)

        case TyFun(ty1=ty1, ty2=ty2, axiom=axiom):
            generate_constraint(ty1, head, state)
            generate_constraint(ty2, head, state)
            if axiom:
                state.add_axiom(unify(node_var(ast), fun_of(node_var(ty1), node_var(ty2))), head)
            else:
                state.add_rule(unify(node_var(ast), fun_of(node_var(ty1), node_var(ty2))), head, ast.id)

        case TyList(ty=ty, axiom=axiom):
            generate_constraint(ty, head, state)
            if axiom:
                state.add_axiom(unify(node_var(ast), list_of(node_var(ty))), head)
            else:
                state.add_rule(unify(node_var(ast), list_of(node_var(ty))), head, ast.id)

        case TyTuple(tys=tys, axiom=axiom):
            if axiom:
                state.add_axiom(unify(node_var(ast), tuple_of(*[node_var(ty) for ty in tys])), head)
            else:
                state.add_rule(unify(node_var(ast), tuple_of(*[node_var(ty) for ty in tys])), head, ast.id)
            for ty in tys:
                generate_constraint(ty, head, state)

        case TyPrefixList():
            state.add_rule(unify(node_var(ast), LAtom(value='list')), head, ast.id)

        case TyPrefixTuple():
            state.add_rule(unify(node_var(ast), LAtom(value='tuple')), head, ast.id)

        case TyPrefixFunction():
            state.add_rule(unify(node_var(ast), LAtom(value='function')), head, ast.id)

        case ExpApp(exp1=exp1, exp2=exp2):
            generate_constraint(exp1, head, state)
            generate_constraint(exp2, head, state)
            fun = fun_of(node_var(exp2), node_var(ast))
            state.add_rule(unify(fun, node_var(exp1)), head, ast.id)

        case ExpLeftSection(left=left, op=op):
            arg = state.fresh()
            result = state.fresh()
            fun = fun_of(node_var(left), arg, result)
            state.add_axiom(unify(fun, node_var(op)), head)
            state.add_rule(unify(node_var(ast), fun_of(arg, result)), head, ast.id)
            generate_constraint(left, head, state)
            generate_constraint(op, head, state)

        case ExpRightSection(right=right, op=op):
            # (==2)
            arg = state.fresh()
            result = state.fresh()
            fun = fun_of(arg, node_var(right), result)
            state.add_axiom(unify(fun, node_var(op)), head)
            state.add_rule(unify(node_var(ast), fun_of(arg, result)), head, ast.id)
            generate_constraint(right, head, state)
            generate_constraint(op, head, state)

        case ExpInfixApp(exp1=exp1, exp2=exp2, op=op):
            fun = fun_of(node_var(exp1), node_var(exp2), node_var(ast))
            state.add_rule(unify(node_var(op), fun), head, ast.id)
            generate_constraint(op, head, state)
            generate_constraint(exp1, head, state)
            generate_constraint(exp2, head, state)

        case ExpLet(binds=decls, exp=exp):
            for decl in decls:
                generate_constraint(decl, head, state)
            generate_constraint(exp, head, state)
            state.add_rule(unify(node_var(ast), node_var(exp)), head, ast.id)

        case ExpIf(cond=cond, if_true=if_ture, if_false=if_false):
            state.add_axiom(unify(node_var(cond), 'p_Bool'), head)
            state.add_rule(unify_all([node_var(ast), node_var(if_false), node_var(if_ture)]), head, ast.id)
            generate_constraint(cond, head, state)
            generate_constraint(if_ture, head, state)
            generate_constraint(if_false, head, state)

        case ExpCase(exp=exp, alts=alts):
            alt_vars = []
            for alt in alts:
                alt: Alt
                pat = alt.pat
                alt_exp = alt.exp
                state.add_axiom(unify(node_var(exp), node_var(pat)), head)
                alt_vars.append(node_var(alt_exp))
                generate_constraint(pat, head, state)
                generate_constraint(alt_exp, head, state)
            state.add_rule(unify_all([node_var(ast), *alt_vars]), head, ast.id)
            generate_constraint(exp, head, state)

        case ExpLambda(exp=exp, pats=pats):
            for pat in pats:
                generate_constraint(pat, head, state)
            fun = fun_of(*[node_var(pat) for pat in pats], node_var(exp))

            state.add_rule(unify(node_var(ast), fun), head, ast.id)
            generate_constraint(exp, head, state)

        case ExpTuple(exps=exps):
            state.add_rule(unify(node_var(ast), tuple_of(*[node_var(exp) for exp in exps])), head, ast.id)
            for exp in exps:
                generate_constraint(exp, head, state)

        case ExpVar(canonical_name=canonical_name) | ExpCon(canonical_name=canonical_name):
            if canonical_name == 'builtin_unit':
                state.add_rule(unify(node_var(ast), 'builtin_Top'), head, ast.id)

            elif canonical_name == 'builtin_bottom':  # Bottom
                pass

            elif canonical_name == head.name:  # Recursive call
                state.add_rule(unify(node_var(ast), 'T'), head, ast.id)

            elif canonical_name == 'builtin_cons' or canonical_name in state.declarations:  # Function
                state.add_rules(state.type_of(canonical_name, node_var(ast), head), head, ast.id)

            else:
                state.add_rule(unify(node_var(ast), LVar(value=f'_{canonical_name}')), head, ast.id)

        case ExpEnumTo(exp=exp) | ExpEnumFrom(exp=exp):
            state.add_rule(unify(node_var(ast), list_of(node_var(exp))), head, ast.id)
            state.add_rule(has_class(node_var(exp), 'p_Enum'), head, ast.id)
            generate_constraint(exp, head, state)

        case ExpEnumFromTo(exp1=exp1, exp2=exp2):
            state.add_rule(unify_all([node_var(ast), list_of(node_var(exp1)), list_of(node_var(exp2))]), head, ast.id)
            state.add_rule(has_class(node_var(exp1), 'p_Enum'), head, ast.id)
            state.add_rule(has_class(node_var(exp2), 'p_Enum'), head, ast.id)

            generate_constraint(exp1, head, state)
            generate_constraint(exp2, head, state)

        case ExpComprehension(exp=exp, quantifiers=quantifiers, guards=guards):
            for quantifier in quantifiers:
                quantifier = cast(Generator, quantifier)
                pat = quantifier.pat
                rhs = quantifier.exp
                state.add_rule(unify(list_of(node_var(pat)), node_var(rhs)), head, ast.id)
                generate_constraint(pat, head, state)
                generate_constraint(rhs, head, state)

            state.add_rule(unify(node_var(ast), list_of(node_var(exp))), head, ast.id)
            generate_constraint(exp, head, state)
            for guard in guards:
                state.add_rule(unify(node_var(guard), LAtom(value='p_Bool')), head, ast.id)
                generate_constraint(guard, head, state)

        case ExpDo(stmts=stmts):
            m = state.fresh()
            a = state.fresh()

            state.add_rule(has_class(m, 'p_Monad'), head, ast.id)
            state.add_rule(unify(node_var(ast), pair(m, a)), head, ast.id)

            for stmt in stmts[0:-1]:
                monad_var = pair(m, wildcard)
                state.add_rule(unify(node_var(stmt), monad_var), head, ast.id)
                generate_constraint(stmt, head, state)

            last_stmt = stmts[-1]
            state.add_rule(unify(node_var(last_stmt), pair(m, a)), head, ast.id)
            generate_constraint(last_stmt, head, state)

        case Generator(pat=pat, exp=exp):
            monad_var = pair(wildcard, node_var(pat))
            state.add_rule(unify(node_var(ast), node_var(exp)), head, ast.id)
            state.add_rule(unify(monad_var, node_var(exp)), head, ast.id)
            generate_constraint(exp, head, state)
            generate_constraint(pat, head, state)

        case Qualifier(exp=exp):
            state.add_rule(unify(node_var(ast), node_var(exp)), head, ast.id)
            generate_constraint(exp, head, state)

        case LetStmt(binds=binds):
            for decl in binds:
                generate_constraint(decl, head, state)

        case LitInt():
            state.add_rule(unify(node_var(ast), 'builtin_Int'), head, ast.id)

        case LitString():
            state.add_rule(unify(node_var(ast), list_of(LAtom(value='builtin_Char'))), head, ast.id)

        case LitChar():
            state.add_rule(unify(node_var(ast), 'builtin_Char'), head, ast.id)

        case LitFrac():
            state.add_rule(unify(node_var(ast), 'builtin_Float'), head, ast.id)
