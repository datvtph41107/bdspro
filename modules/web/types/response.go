// Package types exposes transport-independent web handler capabilities.
package types

// ResponseStatusProvider reports an HTTP status only after response headers
// have been committed. Zero means no response has been sent yet, even when a
// status has been selected for subsequent emission.
//
// This is the same contract as Gitea modules/web/types.ResponseStatusProvider.
type ResponseStatusProvider interface {
	WrittenStatus() int
}
