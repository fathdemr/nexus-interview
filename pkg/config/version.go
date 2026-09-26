package config

// Version is stamped at build time by `make buildFile` via sed.
// When empty (go run / IDE), the config loader treats the binary as a dev build.
var Version = ""
