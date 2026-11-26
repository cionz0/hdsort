package main

// version is the Semantic Version string. Release builds replace it with
// -ldflags "-X main.version=vX.Y.Z". An unset or empty value stays v0.0.0.
var version = "v0.0.0"

// Version returns the build version, or v0.0.0 when the linker left it empty.
func Version() string {
	if version == "" {
		return "v0.0.0"
	}
	return version
}
