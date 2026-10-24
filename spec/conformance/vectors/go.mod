// The conformance vectors are a module of their own so that the go command
// leaves them out of metais.org/metais's module zip (ADR-010 A11). A
// vector's expected tree may hold any path a package can render, and Go's
// module zip refuses some of them, such as an apostrophe; one such path
// would refuse the whole module, and go install with it. Nothing imports
// this module and it holds no Go code.
module metais.org/metais/spec/conformance/vectors

go 1.26.0
