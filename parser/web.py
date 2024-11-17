
from fastapi import FastAPI, Body
from state import InventoryInput, NodeRange, Identifier
from system import static_analysis, parse_modules, run_modules

app = FastAPI()

basic_prelude = '''
data Maybe a = Nothing | Just a
data Either a b = Left a | Right b
data Bool = True | False

class Eq a 
class (Eq a) => Ord a 
instance Eq Int 
instance Eq Float 
instance Eq Bool 
instance Eq Char 
instance Eq a => Eq [a]
instance Ord Int 
instance Ord Float 
instance Ord Char 
instance Ord Bool

(==), (!=) :: Eq a => a -> a -> Bool
(>),(<), (>=), (<=) :: Ord a => a -> a -> Bool
length :: [a] -> Int
id :: a -> a
filter :: (a -> Bool) -> [a] -> [a]
otherwise = True
map :: (a -> b) -> [a] -> [b]
foldr :: (a -> b -> b) -> b -> [a] -> b
foldl :: (b -> a -> b) -> b -> [a] -> b
head :: [a] -> a
tail :: [a] -> [a]
zipWith :: (a->b->c) -> [a] -> [b] -> [c]
fst :: (a,b) -> a
snd :: (a,b) -> b
not :: Bool -> Bool
(||) :: Bool -> Bool -> Bool
(&&) :: Bool -> Bool -> Bool
elem :: a -> [a] -> Bool
class Num a
instance Num Int
instance Num Float
(+),(-),(*) :: Num a => a -> a -> a
mod,div :: Int -> Int -> Int
zip :: [a] -> [b] -> [(a, b)]
(/) :: Float -> Float -> Float
floor :: Num a => a -> Int
ceiling :: Num a => a -> Int
fromIntegral :: Num a => Int -> a
'''

prelude = '''
data Maybe a = Nothing | Just a
data Either a b = Left a | Right b
data Bool = True | False
data IO a = IO a
data Ordering = LT | EQ | GT

class Eq a 
class (Eq a) => Ord a 
instance Eq Int 
instance Eq Float 
instance Eq Bool 
instance Eq Ordering 
instance Eq Char 
instance Eq a => Eq [a]

instance Ord Int 
instance Ord Float 
instance Ord Char 
instance Ord Bool

(==), (/=) :: Eq a => a -> a -> Bool
(>),(<), (>=), (<=) :: Ord a => a -> a -> Bool
compare :: Ord a => a -> a -> Ordering
min,max :: Ord a => a -> a -> a

length :: [a] -> Int
id :: a -> a

class Functor f
class Functor f => Applicative f 
class Applicative m => Monad m 

fmap :: Functor f => (a -> b) -> f a -> f b  
pure :: Applicative f => a -> f a
(<*>) :: Applicative f => f (a -> b) -> f a -> f b
(>>) :: Monad m => m a -> m b -> m b
(>>=) :: Monad m => m a -> (a -> m b) -> m b
return :: Monad m => a -> m a

instance Functor Maybe
instance Functor IO
instance Functor []
instance Functor ((,) a)
instance Functor ((,,) a b)

instance Applicative Maybe
instance Applicative IO
instance Applicative []
instance Applicative ((,) a)
instance Applicative ((,,) a b)

instance Monad Maybe
instance Monad IO
instance Monad []
instance Monad ((,) a)
instance Monad ((,,) a b)

class Monoid a 

instance Monoid [a]

mconcat ::  Monoid a => [a] -> a
mappend :: Monoid a =>  a -> a -> a
mempty :: Monoid a =>  a

filter :: (a -> Bool) -> [a] -> [a]
read :: [Char] -> a
show :: a -> [Char]
otherwise = True
map :: (a -> b) -> [a] -> [b]
foldr :: (a -> b -> b) -> b -> [a] -> b
foldl :: (b -> a -> b) -> b -> [a] -> b
head :: [a] -> a
tail :: [a] -> [a]
zipWith :: (a->b->c) -> [a] -> [b] -> [c]
fst :: (a,b) -> a
snd :: (a,b) -> b
(++) :: [a] -> [a] -> [a]
pi :: Float
not :: Bool -> Bool
const  :: a -> b -> a
reverse :: [a] -> [a]
(||) :: Bool -> Bool -> Bool
(&&) :: Bool -> Bool -> Bool
elem :: a -> [a] -> Bool
even :: Int -> Bool
odd  :: Int -> Bool

class Num a 
instance Num Int
instance Num Float
(+),(-),(*) :: Num a => a -> a -> a
sum :: Num a => [a] -> a
mod,div :: Int -> Int -> Int

class Enum a 
enumFrom ::Enum a =>  a -> [a]
succ :: Enum a => a -> a
pred :: Enum a => a -> a

instance Enum Int
instance Enum Char
instance Enum Bool
instance Enum Float

any :: [Bool] -> Bool
and :: [Bool] -> Bool

zip :: [a] -> [b] -> [(a, b)]
fromIntegral :: Num a => Int -> a
(/) :: Float -> Float -> Float
dropWhile :: (a -> Bool) -> [a] -> [a]
toUpper :: Char -> Char
toLower :: Char -> Char
sqrt :: Float -> Float
(^) :: Num a => a -> Int -> a
floor :: Num a => a -> Int
ceiling :: Num a => a -> Int
($) :: (a -> b) -> a -> b
(.) :: (b -> c) -> (a -> b) -> a -> c
'''

prelude_monad_minimal = '''
data Maybe a = Nothing | Just a
data Bool = True | False

class Monad m

(>>) :: Monad m => m a -> m b -> m b
(>>=) :: Monad m => m a -> (a -> m b) -> m b
return :: Monad m => a -> m a

instance Monad Maybe

'''
no_prelude = ''

@app.post("/translate")
async def translate(body: str = Body()):
    state = run_modules([('Main', body), ('Prelude',  prelude)])
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
        parsing_errors=[NodeRange.from_range(r) for r in state.parsing_errors],
        import_errors=[Identifier.from_buyer(ie) for ie in state.import_errors],
        classes=state.classes,
        type_vars=state.type_vars,
        node_depth=state.node_depth,
        node_graph=[{"parent": parent, "child": child} for parent, child in state.node_graph],
        max_depth=state.max_depth,
        node_range={
            node_id: NodeRange.from_range(_range)
         for node_id, _range in state.node_table.items()},
    )

    return inventory_input


