# Goanna: A Reasonable Haskell Type Checker

Goanna is a Haskell type checker that based on constraint logic. It can suggest alternatives ways to solve a type error. Comparing to traditional Hindley-Milner based type checking, Goanna is less biased and more reliable.

## Key Features
### Precise Error Location

Goanna pinpoints all code fragments that contribute to type errors — not just a single location, as traditional compilers or standalone type checkers often do.

### Fix type errors by traversing the multiverse

Goanna explores all potential solutions to fix type errors, leaving no stone unturned. For each suggested fix, the IDE predicts the resulting type assignments for the program, helping us understand the implications of our choices.
### Type error isolation

If multiple type errors exist, Goanna catches them all, and reports each one separately. It intelligently detects related type errors and consolidates them into a single combined error, reducing the mental overhead for developers.

### Type Inference for Untyped Expressions

Goanna can infer types for both typable and untypable expressions. For traditionally untyped expressions, it assigns them indeterminate types, whose type assignments depend on the programmer's path of action.

## Our approach

Goanna employs its own constraint-based type inference engine rather than relying on the Hindley-Milner type system with algorithm W. In Goanna, Haskell source code is transformed into a constraint set, in the form of annotated ISO Prolog predicates. A Prolog engine then solves the generated constraints to determine the satisfiability of the constraint set, in other word, the type correctness of the program. To identify all potential fixes, Goanna explore Minimal Unsatisfiable Subsets (MUS) and Minimal Correction Subsets (MCS) within the constraint set, and perform a series of analyses base on the result. Each MCS corresponds to a possible fix, while MUS is used to perform type error isolation. 
