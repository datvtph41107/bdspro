package web

// Combo represents a group of method handlers sharing a route pattern.
// A Combo is declared during router registration, not while serving requests.
type Combo struct {
	r       *Router
	pattern string
	h       []any
}

func (c *Combo) Get(h ...any) *Combo {
	c.r.Get(c.pattern, append(c.h, h...)...)
	return c
}

func (c *Combo) Post(h ...any) *Combo {
	c.r.Post(c.pattern, append(c.h, h...)...)
	return c
}

func (c *Combo) Delete(h ...any) *Combo {
	c.r.Delete(c.pattern, append(c.h, h...)...)
	return c
}

func (c *Combo) Put(h ...any) *Combo {
	c.r.Put(c.pattern, append(c.h, h...)...)
	return c
}

func (c *Combo) Patch(h ...any) *Combo {
	c.r.Patch(c.pattern, append(c.h, h...)...)
	return c
}
