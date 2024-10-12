from __future__ import annotations

from pydantic import BaseModel

class LTerm(BaseModel):
    pass


class LVar(LTerm):
    value: str

    def __str__(self):
        return self.value


class LAtom(LTerm):
    value: str

    def __str__(self):
        return self.value


class LStruct(LTerm):
    functor: str
    args: list[LTerm]

    def __str__(self):
        if self.functor == '=':
            return f'{self.args[0]} = {self.args[1]}'
        else:
            return f'{self.functor}({", ".join([arg.__str__() for arg in self.args])})'


class LList(LTerm):
    elements: list[LTerm]

    def __str__(self):
        return f'[{", ".join([elem.__str__() for elem in self.elements])}]'

# Special Vars
T = LVar(value="T")
Call = LVar(value="Calls")
Call_ = LVar(value="Calls_")
GammaVar = LVar(value="Gamma")
GammaVar_ = LVar(value="Gamma_")
ZetaVar = LVar(value="Zeta")
ZetaVar_ = LVar(value="Zeta_")
Classes = LVar(value="Classes")

# Special terms
succeed = LAtom(value="true")
fail = LAtom(value="false")
nil = LAtom(value="nil")
wildcard = LVar(value="_")
cut = LAtom(value="!")


# Special functors
def cons(x: LTerm, xs: LTerm) -> LTerm:
    return LStruct(functor="[|]", args=[x, xs])


# Special predicates
def unify(a: LTerm | str, b: LTerm | str):
    if isinstance(a, str):
        if a.islower():
            a = LAtom(value=a)
        else:
            a = LVar(value=a)
    if isinstance(b, str):
        if b.islower():
            b = LAtom(value=b)
        else:
            b = LVar(value=b)
    return LStruct(functor='eq', args=[a, b])


def unify_all(terms: list[LTerm | str]):
    return LStruct(functor="all_equal", args=[LList(elements=terms)])


def once(term: LTerm) -> LTerm:
    return LStruct(functor="once", args=[term])

