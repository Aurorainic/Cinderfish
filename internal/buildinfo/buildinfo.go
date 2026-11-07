// Package buildinfo holds values compiled into the binary. They live here so
// the release version is bumped in one place: meta, NodeInfo and the
// outbound User-Agent all read the same constant.
package buildinfo

// Version is the Cinderfish release version.
const Version = "0.1.2"
