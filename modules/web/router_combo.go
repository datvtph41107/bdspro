package web

// Combo collects methods on one path and retains a shared middleware prefix.
type Combo struct {
	r    *Router
	path string
	base []any
}

func (c *Combo) Get(h ...any) *Combo {
	c.r.Get(c.path, append(append([]any(nil), c.base...), h...)...)
	return c
}
func (c *Combo) Post(h ...any) *Combo {
	c.r.Post(c.path, append(append([]any(nil), c.base...), h...)...)
	return c
}
func (c *Combo) Put(h ...any) *Combo {
	c.r.Put(c.path, append(append([]any(nil), c.base...), h...)...)
	return c
}
func (c *Combo) Patch(h ...any) *Combo {
	c.r.Patch(c.path, append(append([]any(nil), c.base...), h...)...)
	return c
}
func (c *Combo) Delete(h ...any) *Combo {
	c.r.Delete(c.path, append(append([]any(nil), c.base...), h...)...)
	return c
}
