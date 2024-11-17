from typing import cast, Any
from parser.syntax import *
from state import ParseEnv

Synonym = dict[str, tuple[list[str], Ty]]

def gather_synonyms(_ast: Pretty) -> Synonym:
    match _ast:
        case Module(decls=decls):
            obj = {}
            for decl in decls:
                obj.update(gather_synonyms(decl))
            return obj
        case TypeDecl(d_head=d_head, ty=rhs):
            con_name: str = d_head.name
            ty_vars: list[TyVar] = d_head.ty_vars
            return {con_name: ([v.name for v in ty_vars], rhs)}

        case _:
            return {}


def merge_synonyms(_synonyms: list[Synonym]) -> Synonym:
    return {k: v for synonym in _synonyms for k, v in synonym.items()}

def collapse_type_metadata(input_type: Ty, env: ParseEnv, new_loc: Any, top_level: bool) -> Ty:
    match input_type:
        case TyVar(name=name, canonical_name=canonical_name):
            return TyVar(name=name, id=env.new_id(), loc=new_loc, canonical_name=canonical_name)
        case TyCon(name=name, module=module, canonical_name=canonical_name):
            return TyCon(name=name, module=module, canonical_name=canonical_name, id=env.new_id(), loc=new_loc, axiom=not top_level)
        case TyApp(ty1=ty1, ty2=ty2):
            return TyApp(ty1=ty1, ty2=ty2, id=env.new_id(), loc=new_loc, axiom=not top_level)
        case TyTuple(tys=tys):
            return TyTuple(tys=[collapse_type_metadata(t, env, new_loc, False) for t in tys], id=env.new_id(), loc=new_loc, axiom=not top_level)
        case TyList(ty=ty):
            return TyList(ty=collapse_type_metadata(ty, env, new_loc, False), id=env.new_id(), loc=new_loc, axiom=not top_level)
        case TyForall(context=context, ty=ty):
            return TyForall(context=context, ty=collapse_type_metadata(ty, env, new_loc, False), id=env.new_id(), loc=new_loc, axiom=not top_level)
        case _:
            raise Exception("Unknown type of Ty")


def replace(input_type: Ty, replacements: list[tuple[str, Ty]], env: ParseEnv, top_level: bool =False) -> Ty:
    match input_type:
        case TyVar(name=name):
            for k, v in replacements:
                if name == k:
                    return v.__class__(**{k: v for k, v in v.__dict__.items() if k != 'id'}, id=env.new_id(), axiom=not top_level)
            return TyVar(name=name, loc=input_type.loc, id=env.new_id(), canonical_name=input_type.canonical_name, axiom=not top_level)
        case TyApp(ty1=ty1, ty2=ty2):
            return TyApp(ty1=replace(ty1, replacements, env), ty2=replace(ty2, replacements, env),
                         loc=input_type.loc, id=env.new_id(), axiom=not top_level)
        case TyCon():
            return TyCon(name=input_type.name,
                         loc=input_type.loc,
                         id=env.new_id(),
                         canonical_name=input_type.canonical_name,
                         module=input_type.module,
                         axiom=not top_level)
        case TyTuple(tys=tys):
            return TyTuple(tys=[replace(ty, replacements, env) for ty in tys], loc=input_type.loc, id=env.new_id(), axiom=not top_level)
        case TyFun(ty1=ty1, ty2=ty2):
            return TyFun(ty1=replace(ty1, replacements, env), ty2=replace(ty2, replacements, env),
                         loc=input_type.loc, id=env.new_id(), axiom=not top_level)
        case TyList(ty=ty):
            return TyList(ty=replace(ty, replacements, env), loc=input_type.loc, id=env.new_id(), axiom=not top_level)
        case TyForall(ty=ty):
            return TyForall(ty=replace(ty, replacements, env), loc=input_type.loc, context=input_type.context,
                            id=env.new_id(), axiom=not top_level)
        case _:
            raise Exception("Unknown type of Ty")


def replace_synonyms(_synonyms: Synonym, input_type: Pretty, env: ParseEnv) -> tuple[bool, Pretty]:
    def unroll_type(_input_type: Ty) -> list[Ty]:
        match _input_type:
            case TyApp(ty1=ty1, ty2=ty2):
                return unroll_type(ty1) + unroll_type(ty2)
            case _:
                return [_input_type]

    synonym_names = _synonyms.keys()

    match input_type:
        case TyCon(name=name):
            if name in synonym_names:
                if _synonyms[name][0]:
                    raise Exception("The kind of synonym does not match its usage")
                ty = _synonyms[name][1]
                new_ty = collapse_type_metadata(ty, env, input_type.loc, True)
                return True, new_ty
            else:
                return False, input_type

        case TyApp():
            types = unroll_type(cast(TyApp, input_type))
            if isinstance(types[0], TyCon) and types[0].name in synonym_names:
                replace_from = _synonyms[types[0].name][0]
                replace_to = types[1:]
                replacing_type = _synonyms[types[0].name][1]
                if len(replace_from) != len(replace_to):
                    raise Exception("The kind of synonym does not match its usage")
                return True, replace(replacing_type, list(zip(replace_from, replace_to)), env, True)

            else:
                input_type: TyApp
                replaced1, new_type1 = replace_synonyms(_synonyms, input_type.ty1, env)
                replaced2, new_type2 = replace_synonyms(_synonyms, input_type.ty2, env)
                return replaced1 or replaced2, TyApp(ty1=cast(Ty, new_type1), ty2=cast(Ty, new_type2),
                                                     loc=input_type.loc, id=input_type.id, axiom=input_type.axiom)

        case _:
            obj = {}
            replaced = False
            for k, v in input_type.__dict__.items():
                if isinstance(v, Pretty):
                    replaced, new_ast = replace_synonyms(_synonyms, v, env)
                    obj = {**obj, k: new_ast}
                elif isinstance(v, list):
                    if len(v) == 0:
                        obj = {**obj, k: v}
                    elif isinstance(v[0], Pretty):
                        each_replaced, new_ast = list(zip(*[replace_synonyms(_synonyms, vi, env) for vi in v]))
                        replaced = any(each_replaced)
                        obj = {**obj, k: list(new_ast)}
                    else:
                        obj = {**obj, k: v}

                else:
                    obj = {**obj, k: v}
            return replaced, type(input_type)(**obj)


def remove_synonyms(input_type: Module) -> Module:
    return Module(name=input_type.name, loc=input_type.loc, imports=input_type.imports,
                  decls=[decl for decl in input_type.decls if not isinstance(decl, TypeDecl)], id=input_type.id)


def replace_synonyms_recursive(_synonyms: Synonym, input_type: Module, env: ParseEnv):
    loop_count = 0
    replaced = True
    while replaced:
        if loop_count > 50:
            raise Exception("Possible cyclic definition in type synonym")
        loop_count += 1
        replaced, new_ast = replace_synonyms(_synonyms, input_type, env)
        input_type = new_ast

    return remove_synonyms(input_type)


def translate_synonyms(modules: list[Module], env: ParseEnv) -> list[Module]:
    synonyms = merge_synonyms([gather_synonyms(module) for module in modules])
    return [replace_synonyms_recursive(synonyms, module, env) for module in modules]