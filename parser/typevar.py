from state import *
from parser.traverse import Traverse
from parser.syntax import *


class TypeVar(BaseModel):
    decl: str
    var_name: str
    type_classes: set[str]


class GatherTypeVar(BaseModel):
    current_decls: list[str]
    type_vars: TypeVars


def update_type_vars(data: GatherTypeVar, ast: Pretty, *_) -> GatherTypeVar:
    match ast:
        case ClassDecl(d_head=d_head, decls=decls):
            d_head: DeclHead
            class_name = d_head.canonical_name
            type_var_name = d_head.ty_vars[0].name
            for decl in decls:
                if not isinstance(decl, TypeSig):
                    continue
                names = decl.canonical_names
                for name in names:
                    if decl in data.type_vars and type_var_name in data.type_vars[name]:
                        data.type_vars[name][type_var_name].add(class_name)
                    else:
                        data.type_vars[name][type_var_name] = {class_name}

        case TypeSig(canonical_names=canonical_names):
            data.current_decls = canonical_names

        case TyForall(context=Context(assertions=assertions)):
            for assertion in assertions:
                assertion: TyApp
                class_name = assertion.ty1.canonical_name
                type_var_name = assertion.ty2.name
                for decl in data.current_decls:
                    if decl in data.type_vars:
                        if type_var_name in data.type_vars[decl]:
                            data.type_vars[decl][type_var_name].add(class_name)
                        else:
                            data.type_vars[decl][type_var_name] = {class_name}
                    else:
                        data.type_vars.setdefault(decl, {}).setdefault(type_var_name, {class_name})

        case TyVar(name=name):
            for decl in data.current_decls:
                if decl in data.type_vars:
                    if name in data.type_vars[decl]:
                        continue
                    else:
                        data.type_vars[decl][name] = set()
                else:
                    data.type_vars.setdefault(decl, {}).setdefault(name, set())
    return data


def gather_type_vars(asts: list[Pretty], classes: SuperClasses) -> TypeVars:
    traverser = Traverse(
        init_data=GatherTypeVar(current_decls=[], type_vars={}),
        update_func=update_type_vars)
    traverser.traverse_all(asts)
    data: GatherTypeVar = traverser.value
    type_vars = data.type_vars
    for decl, var_class in type_vars.items():
        for type_var, type_var_classes in var_class.items():
            type_vars[decl][type_var] = type_var_classes.union(*[classes[c] for c in type_var_classes])
    return data.type_vars
