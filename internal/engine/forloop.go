package engine

// ForloopDrop holds the forloop metadata available as {{ forloop.index }}, etc.
// Defined in the engine package so Context can hold a direct pointer to it,
// enabling an O(1) fast path in VariableLookup that bypasses the scope scan.
type ForloopDrop struct {
	Index   int
	Index0  int
	Rindex  int
	Rindex0 int
	First   bool
	Last    bool
	Length  int
}
