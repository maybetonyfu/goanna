from typing import Callable, TypeVar

from parser.syntax import *

V = TypeVar('V')

IdFunc = Callable[[], int]


class Traverse:
    def __init__(self, init_data: V, update_func: Callable[[V, Pretty, Pretty | None], V]):
        super().__init__()
        self.data = init_data
        self.update = update_func
        self.cleanup = lambda data, ast, parent: data

    def set_cleanup(self, cleanup_function: Callable[[V, Pretty, Pretty | None], V]):
        self.cleanup = cleanup_function

    @property
    def value(self):
        return self.data

    def traverse_all(self, asts: list[Pretty]):
        for ast in asts:
            self.traverse(ast)

    def traverse(self, ast: Pretty, parent: Pretty = None) -> Pretty:

        self.data = self.update(self.data, ast, parent)

        match ast:
            case Module(decls=decls):
                ast.decls = [self.traverse(decl, ast) for decl in decls]

            # Declarations
            case PatBind(pat=pat, rhs=rhs):
                ast.pat = self.traverse(pat, ast)
                ast.rhs = self.traverse(rhs, ast)

            case TypeSig(ty=ty):
                ast.ty = self.traverse(ty, ast)
            case ClassDecl(context=context, d_head=d_head, decls=decls):
                ast.decls = [self.traverse(decl, ast) for decl in decls]

                if context is not None:
                    ast.context = self.traverse(context, ast)
                ast.d_head = self.traverse(d_head, ast)
            case InstDecl(context=context, tys=tys, body=body):
                ast.body = [self.traverse(decl, ast) for decl in body]

                if context is not None:
                    ast.context = self.traverse(context, ast)
                ast.tys = [self.traverse(ty, ast) for ty in tys]

            case DataDecl(d_head=d_head, constructors=constructors, deriving=deriving):
                ast.d_head = self.traverse(d_head, ast)
                ast.constructors = [self.traverse(c, ast) for c in constructors]
                ast.deriving = [self.traverse(ty, ast) for ty in deriving]

            # Patterns
            case PVar() | PWildCard():
                pass
            case PApp(pats=pats):
                ast.pats = [self.traverse(pat, ast) for pat in pats]

            case PList(pats=pats):
                ast.pats = [self.traverse(pat, ast) for pat in pats]

            case PTuple(pats=pats):
                ast.pats = [self.traverse(pat, ast) for pat in pats]

            case PInfix(pat1=pat1, pat2=pat2):
                ast.pat1 = self.traverse(pat1, ast)
                ast.pat2 = self.traverse(pat2, ast)

            # Expressions
            case ExpVar() | ExpCon():
                pass

            case ExpApp(exp1=exp1, exp2=exp2):
                ast.exp1 = self.traverse(exp1, ast)
                ast.exp2 = self.traverse(exp2, ast)

            case ExpInfixApp(exp1=exp1, exp2=exp2):
                ast.exp1 = self.traverse(exp1, ast)
                ast.exp2 = self.traverse(exp2, ast)

            case ExpLambda(exp=exp, pats=pats):
                ast.exp = self.traverse(exp, ast)
                ast.pats = [self.traverse(pat, ast) for pat in pats]

            case ExpLet(binds=decls, exp=exp):
                ast.binds = [self.traverse(decl, ast) for decl in decls]

                ast.exp = self.traverse(exp, ast)
            case ExpIf(cond=cond, if_true=then, if_false=else_):
                ast.cond = self.traverse(cond, ast)
                ast.if_true = self.traverse(then, ast)
                ast.if_false = self.traverse(else_, ast)
            case ExpDo(stmts=stmts):
                ast.stmts = [self.traverse(stmt, ast) for stmt in stmts]

            case ExpCase(exp=exp, alts=alts):
                ast.exp = self.traverse(exp, ast)
                ast.alts = [self.traverse(alt, ast) for alt in alts]

            case ExpTuple(exps=exps):
                ast.exps = [self.traverse(exp, ast) for exp in exps]

            case ExpList(exps=exps):
                ast.exps = [self.traverse(exp, ast) for exp in exps]

            case ExpLeftSection(left=exp, op=op):
                ast.left = self.traverse(exp, ast)
                ast.op = self.traverse(op, ast)
            case ExpRightSection(op=op, right=exp):
                ast.right = self.traverse(exp, ast)
                ast.op = self.traverse(op, ast)
            case ExpEnumFrom(exp=exp):
                ast.exp = self.traverse(exp, ast)
            case ExpEnumTo(exp=exp):
                ast.exp = self.traverse(exp, ast)
            case ExpEnumFromTo(exp1=exp1, exp2=exp2):
                ast.exp1 = self.traverse(exp1, ast)
                ast.exp2 = self.traverse(exp2, ast)

            # Statements
            case Generator(exp=exp, pat=pat):
                ast.exp = self.traverse(exp, ast)
                ast.pat = self.traverse(pat, ast)
            case Qualifier(exp=exp):
                ast.exp = self.traverse(exp, ast)
            case LetStmt(binds=decls):
                ast.binds = [self.traverse(decl, ast) for decl in decls]

            # Types
            case TyVar() | TyCon() | TyPrefixList() | TyPrefixTuple():
                pass
            case TyApp(ty1=ty1, ty2=ty2):
                ast.ty1 = self.traverse(ty1, ast)
                ast.ty2 = self.traverse(ty2, ast)
            case TyFun(ty1=ty1, ty2=ty2):
                ast.ty1 = self.traverse(ty1, ast)
                ast.ty2 = self.traverse(ty2, ast)
            case TyTuple(tys=tys):
                ast.tys = [self.traverse(ty, ast) for ty in tys]

            case TyList(ty=ty):
                ast.ty = self.traverse(ty, ast)
            case TyForall(context=context, ty=ty):
                if context is not None:
                    ast.context = self.traverse(context, ast)
                ast.ty = self.traverse(ty, ast)

            # Literals
            case LitInt(_) | LitFrac(_) | LitString(_) | LitChar(_):
                pass

            # Misc
            case DataCon(tys=tys):
                ast.tys = [self.traverse(ty, ast) for ty in tys]

            case UnguardedRhs(exp=exp, wheres=wheres):
                ast.exp = self.traverse(exp, ast)
                ast.wheres = [self.traverse(where, ast) for where in wheres]

            case GuardedRhs(branches=branches, wheres=wheres):
                ast.branches = [self.traverse(b, ast) for b in branches]

                ast.wheres = [self.traverse(where, ast) for where in wheres]

            case GuardBranch(guards=guards, exp=exp):
                ast.guards = [self.traverse(guard, ast) for guard in guards]

                ast.exp = self.traverse(exp, ast)
            case Context(assertions=assertions):
                ast.assertions = [self.traverse(assertion, ast) for assertion in assertions]

            case DeclHead(ty_vars=ty_vars):
                ast.ty_vars = [self.traverse(ty_var, ast) for ty_var in ty_vars]

            case Alt(pat=pat, exp=exp, binds=binds):
                ast.pat = self.traverse(pat, ast)
                ast.exp = self.traverse(exp, ast)
                ast.binds = [self.traverse(bind, ast) for bind in binds]

        self.data = self.cleanup(self.data, ast, parent)
        return ast
