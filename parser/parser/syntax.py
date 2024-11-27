from __future__ import annotations

from dataclasses import dataclass, field
from typing import Union

from state import Range


@dataclass
class Pretty:
    id: int = field(repr=False)
    loc: Range = field(repr=False)

    def pretty(self):
        obj = {'type': self.__class__.__name__}
        for k, v in self.__dict__.items():
            if k == 'loc':
                obj = {**obj, k: v}
            # elif k == 'id':
            #     continue
            elif v is None:
                obj = {**obj, k: v}
            elif isinstance(v, str):
                obj = {**obj, k: v}
            elif isinstance(v, int):
                obj = {**obj, k: v}
            elif isinstance(v, list):
                for x in v:
                    if isinstance(x, Pretty):
                        obj = {**obj, k: [x.pretty() for x in v]}
                    else:
                        obj = {**obj, k: v}
            elif isinstance(v, Pretty):
                obj = {**obj, k: v.pretty()}
        return obj


Pat = Union['PWildCard', 'PApp', 'Lit',  # Simplify PLit to Lit
'PList', 'PTuple', 'PVar', 'PInfix'
]

Rhs = Union["UnguardedRhs", "GuardedRhs"]

Lit = Union['LitChar', 'LitString', 'LitInt', 'LitFrac']

Ty = Union['TyCon', 'TyApp', 'TyFun', 'TyTuple', 'TyList', 'TyVar', 'TyForall', 'TyPrefixList', 'TyPrefixTuple', "TyPrefixFunction"]

Exp = Union['ExpVar', 'ExpCon', 'Lit',  # 'ExpLit', Use lit to simplify
'ExpApp', 'ExpInfixApp',  # Added this
'ExpLambda', 'ExpLet', 'ExpIf', 'ExpCase', 'ExpDo',
'ExpTuple', 'ExpList', 'ExpLeftSection', 'ExpRightSection',
'ExpEnumFrom', 'ExpEnumTo', 'ExpEnumFromTo',
'ExpComprehension'
]

Stmt = Union['Generator', 'Qualifier', 'LetStmt']

Decl = Union['TypeDecl', 'DataDecl', 'ClassDecl', 'InstDecl', 'TypeSig', # 'FunBind',
'PatBind']


@dataclass
class Module(Pretty):
    name: str
    decls: list[Decl]
    imports: list[str]


# Declarations
@dataclass
class TypeDecl(Pretty):
    d_head: DeclHead
    ty: Ty


@dataclass
class DataDecl(Pretty):
    d_head: DeclHead
    constructors: list[DataCon]
    deriving: list[TyCon]


@dataclass
class ClassDecl(Pretty):
    context: Context | None
    d_head: DeclHead
    decls: list[Decl]


@dataclass
class InstDecl(Pretty):
    context: Context | None
    name: str
    module: str | None
    canonical_name: str | None
    tys: list[Ty]
    body: list[Decl]


@dataclass
class PatBind(Pretty):
    pat: Pat
    rhs: Rhs
    # rhs_list: list[Rhs]  # wheres: list[Decl] Move the were clauses to Rhs


@dataclass
class TypeSig(Pretty):
    names: list[str]
    canonical_names: list[str]
    ty: Ty


# Statements
@dataclass
class Generator(Pretty):
    pat: Pat
    exp: Exp


@dataclass
class Qualifier(Pretty):
    exp: Exp


@dataclass
class LetStmt(Pretty):
    binds: list[Decl]


# Patterns
@dataclass
class PWildCard(Pretty):
    pass


@dataclass
class PApp(Pretty):
    name: str
    module: str | None
    canonical_name: str | None
    pats: list[Pat]


@dataclass
class PList(Pretty):
    pats: list[Pat]


@dataclass
class PTuple(Pretty):
    pats: list[Pat]


@dataclass
class PVar(Pretty):
    name: str
    canonical_name: str | None


@dataclass
class PInfix(Pretty):
    pat1: Pat
    name: str
    module: str | None
    canonical_name: str | None
    pat2: Pat


# Literals

@dataclass
class LitChar(Pretty):
    pass


@dataclass
class LitString(Pretty):
    pass


@dataclass
class LitInt(Pretty):
    pass


@dataclass
class LitFrac(Pretty):
    pass


# Types
@dataclass
class TyCon(Pretty):
    name: str
    module: str | None
    canonical_name: str | None
    axiom: bool


@dataclass
class TyApp(Pretty):
    ty1: Ty
    ty2: Ty
    axiom: bool


@dataclass
class TyFun(Pretty):
    ty1: Ty
    ty2: Ty
    axiom: bool


@dataclass
class TyTuple(Pretty):
    tys: list[Ty]
    axiom: bool


@dataclass
class TyList(Pretty):
    ty: Ty
    axiom: bool


@dataclass
class TyPrefixList(Pretty):
    pass

@dataclass
class TyPrefixTuple(Pretty):
    arity: int

@dataclass()
class TyPrefixFunction(Pretty):
    pass

@dataclass
class TyVar(Pretty):
    name: str
    canonical_name: str | None
    axiom: bool


@dataclass
class TyForall(Pretty):
    context: Context | None
    ty: Ty
    axiom: bool


# Expressions
@dataclass
class ExpVar(Pretty):
    name: str
    module: str | None
    canonical_name: str | None


@dataclass
class ExpCon(Pretty):
    name: str
    module: str | None
    canonical_name: str | None


@dataclass
class ExpApp(Pretty):
    exp1: Exp
    exp2: Exp


@dataclass
class ExpInfixApp(Pretty):
    exp1: Exp
    op: ExpVar
    exp2: Exp

@dataclass
class ExpLambda(Pretty):
    pats: list[Pat]
    exp: Exp


@dataclass
class ExpLet(Pretty):
    binds: list[Decl]
    exp: Exp


@dataclass
class ExpIf(Pretty):
    cond: Exp
    if_true: Exp
    if_false: Exp


@dataclass
class ExpDo(Pretty):
    stmts: list[Stmt]


@dataclass
class ExpCase(Pretty):
    exp: Exp
    alts: list[Alt]


@dataclass
class ExpTuple(Pretty):
    exps: list[Exp]


@dataclass
class ExpList(Pretty):
    exps: list[Exp]


@dataclass
class ExpLeftSection(Pretty):
    left: Exp
    op: Exp


@dataclass
class ExpRightSection(Pretty):
    op: Exp
    right: Exp


@dataclass
class ExpEnumFromTo(Pretty):
    exp1: Exp
    exp2: Exp


@dataclass
class ExpEnumFrom(Pretty):
    exp: Exp


@dataclass
class ExpEnumTo(Pretty):
    exp: Exp

@dataclass
class ExpComprehension(Pretty):
    exp: Exp
    quantifiers: list[Generator]
    guards: list[Exp]

# Misc
@dataclass
class Alt(Pretty):
    pat: Pat
    exp: Exp
    binds: list[Decl]


@dataclass
class UnguardedRhs(Pretty):
    exp: Exp
    wheres: list[Decl]


@dataclass
class GuardedRhs(Pretty):
    branches: list[GuardBranch]
    wheres: list[Decl]


@dataclass
class GuardBranch(Pretty):
    guards: list[Exp]
    exp: Exp


@dataclass
class DataCon(Pretty):
    name: str
    canonical_name: str | None
    tys: list[Ty]


@dataclass
class DeclHead(Pretty):
    name: str
    canonical_name: str | None
    ty_vars: list[TyVar]


@dataclass
class Context(Pretty):
    assertions: list[Ty]
