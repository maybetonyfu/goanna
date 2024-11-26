from collections import defaultdict
from typing import Literal, Any

from pydantic import BaseModel

from logic import LTerm

Point = tuple[int, int]
Range = tuple[Point, Point]


def before(p1: Point, p2: Point) -> bool:
    return p1[0] < p2[0] or (p1[0] == p2[0] and p1[1] <= p2[1])


def after(p1: Point, p2: Point) -> bool:
    return p1[0] > p2[0] or (p1[0] == p2[0] and p1[1] >= p2[1])


def within(loc1: Range, loc2: Range) -> bool:
    return after(loc1[0], loc2[0]) and before(loc1[1], loc2[1])


class HaskellParsingError(Exception):
    def __init__(self, loc: Range):
        super().__init__(f'Parsing error at {loc}')
        self.loc = loc


class ParseEnv:
    def __init__(self):
        self.counter = 0

    def new_id(self) -> int:
        self.counter += 1
        return self.counter


class EffectiveRange(BaseModel):
    excludes: list[Range]
    is_global: bool = False
    ranges: Range | None = None


class Vendor(BaseModel):
    node_id: int
    name: str
    canonical_name: str
    type: Literal['term', 'type']
    is_declaration: bool = False
    module: str
    effective_range: EffectiveRange


class Buyer(BaseModel):
    node_id: int
    name: str
    type: Literal['term', 'type']
    module: str | None
    usage_module: str
    usage_loc: Range
    canonical_name: str | None


type ArgName = str
type DeclaredName = str

type ChildDecl = str
type ParentDecl = str

type Arguments = dict[DeclaredName, list[ArgName]]
type Closures = dict[ChildDecl, list[ParentDecl]]

type ClassName = str
type SuperClassName = str
type SuperClasses = dict[ClassName, set[SuperClassName]]

type TypeVarName = str
type TypeVars = dict[DeclaredName, dict[TypeVarName, set[ClassName]]]


class RuleHead(BaseModel):
    type: Literal['type', 'instance']
    name: str
    module: str
    id: int | None

    def __str__(self):
        return self.name

class Rule(BaseModel):
    head: RuleHead
    body: LTerm
    node_id: int | None
    axiom: bool = False
    id: int | None = None

    def __str__(self):
        return f'{self.head} :- {self.body}'


class NodeRange(BaseModel):
    from_line: int
    to_line: int
    from_col: int
    to_col: int

    @classmethod
    def from_range(cls, range: Range):
        return cls(from_line=range[0][0], to_line=range[1][0], from_col=range[0][1], to_col=range[1][1])

class Identifier(BaseModel): # This is just a simpler vendor, for sending through wire
    node_id: int
    name: str
    node_range: NodeRange
    is_type: bool
    is_term: bool

    @classmethod
    def from_buyer(cls, buyer: Buyer):
        return cls(
            node_id=buyer.node_id,
            name=buyer.name,
            node_range=NodeRange.from_range(buyer.usage_loc),
            is_type=buyer.type == "type",
            is_term=buyer.type == "term"
        )


class InventoryInput(BaseModel):
    declarations: list[str]
    top_levels: list[str]
    base_modules: list[str]
    rules: list[dict] = []
    arguments: Arguments = {}
    classes: SuperClasses = {}
    type_vars: dict = {}
    node_depth: dict[int, int] = {}
    node_graph: list[dict] = []
    max_depth: int = 0
    collectors: dict[str, list[str]]
    node_range: dict[int, NodeRange] = {}
    parsing_errors: list[NodeRange]
    import_errors: list[Identifier]

class State(BaseModel):
    asts: list[Any] = []
    text_lines: dict[str, list[str]] = {}
    vendors: list[Vendor] = []
    buyers: list[Buyer] = []
    parsing_errors: list[Range] = []
    import_errors: list[Buyer] = []
    module_mapping: dict[str, str] = {}
    declarations: list[str] = []
    top_levels: list[str] = []
    rules: list[Rule] = []
    arguments: Arguments = {}
    closures: Closures = {}
    classes: SuperClasses = {}
    collectors: dict[str, list[str]] = defaultdict(list)
    type_vars: TypeVars = {}
    node_depth: dict[int, int] = {}
    node_graph: list[tuple[int, int]] = []
    node_table: dict[int, Range] = {}
    max_depth: int = 0

    def get_declarations(self) -> list[str]:
        return self.declarations

    def add_rule(self, rule: Rule):
        rule.id = rule.node_id
        self.rules.append(rule)

    def max_level(self) -> int:
        return self.max_depth

    def is_parent_of(self, parent: str, child: str) -> bool:
        if child not in self.closures:
            return False
        return parent in self.closures[child]

    def add_class_var(self, head_name: str, class_var:str):
        self.collectors[head_name].append(class_var)