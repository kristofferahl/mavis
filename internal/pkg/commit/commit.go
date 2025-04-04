package commit

type Commit struct {
	Type         string
	Scope        string
	Description  string
	Breaking     bool
	OptionalBody string
}

func (c Commit) Summary() string {
	str := c.Type
	if c.Scope != "" {
		str += "(" + c.Scope + ")"
	}
	if c.Breaking {
		str += "!"
	}
	str += ": "
	str += c.Description
	return str
}

func (c Commit) Body() string {
	str := ""
	if c.Breaking {
		str += "BREAKING CHANGE: "
	}
	str += c.OptionalBody
	return str
}

func (c Commit) HasBody() bool {
	return c.OptionalBody != ""
}

func (c Commit) String() string {
	str := c.Summary() + "\n"
	if c.HasBody() {
		str += "\n" + c.Body() + "\n"
	}
	return str
}
