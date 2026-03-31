package rename

import "goanna/haskell/parser"

// RenameAll generates identifiers for all modules, renames declaration sites,
// and resolves all name references. All operations mutate the modules in place.
func RenameAll(modules []*parser.Module) {
	moduleValues := make([]parser.Module, len(modules))
	for i, m := range modules {
		moduleValues[i] = *m
	}

	env := &RenameEnv{}
	result := env.GenIdentifiersAll(moduleValues)

	for _, m := range modules {
		RenameDecl(m, result)
		RenameTypeDecl(m, result)
	}

	ResolveAll(modules, result)
	RenameTypeVars(modules)
}
