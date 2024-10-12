
from fastapi import FastAPI, Body
from fastapi.middleware.cors import CORSMiddleware
import time
from starlette.staticfiles import StaticFiles
from state import InventoryInput, NodeRange
from system import static_analysis, parse_modules, run_modules

app = FastAPI()


prelude = '''
data Maybe a = Nothing | Just a
data Either a b = Left a | Right b
(+),(-),(*),mod,div :: Int -> Int -> Int
data Bool = True | False
class Eq a where
class (Eq a) => Ord a where
instance Eq Int where
instance Eq Float where
instance Eq Bool where
instance Ord Int where
instance Ord Float where
(==), (!=) :: Eq a => a -> a -> Bool
(>),(<), (>=), (<=) :: Ord a => a -> a -> Bool
'''

no_prelude = ''

@app.post("/translate")
async def translate(body: str = Body()):
    state = run_modules([('Main', body), ('Prelude',  no_prelude)])

    inventory_input = InventoryInput(
        base_modules=["Prelude"],
        declarations=state.declarations,
        rules=[{
            'head': r.head.model_dump(),
            'id': r.id,
            'axiom': r.axiom,
            'body': str(r.body),
        } for r in state.rules],
        arguments=state.arguments,
        classes=state.classes,
        type_vars=state.type_vars,
        node_depth=state.node_depth,
        node_graph=[{"parent": parent, "child": child} for parent, child in state.node_graph],
        max_depth=state.max_depth,
        node_range={
            node_id: NodeRange(from_line=range[0][0], to_line=range[1][0], from_col=range[0][1], to_col=range[1][1])
         for node_id, range in state.node_table.items()},
    )

    return inventory_input


