package structs

// LoginPasswordOption is the HTTP contract for local credential login.
// Validation applies before the handler invokes any credential lookup.
type LoginPasswordOption struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=1024"`
}
