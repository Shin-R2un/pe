package cli

// reserved is the set of words that cannot be used as a snippet key,
// and that the dispatcher recognizes as subcommands.
var reserved = map[string]struct{}{
	"a": {}, "add": {},
	"l": {}, "list": {}, "ls": {},
	"s": {}, "search": {}, "find": {},
	"e": {}, "edit": {},
	"d": {}, "delete": {}, "rm": {},
	"?": {}, "show": {},
	"help": {}, "version": {},
	"completion": {}, "__complete": {},
	"-h": {}, "--help": {},
	"-v": {}, "--version": {},
}

// isReserved reports whether the word is reserved.
func isReserved(s string) bool {
	_, ok := reserved[s]
	return ok
}
