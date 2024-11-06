package hh

type Area struct {
	ID   string
	Name string
}

type area struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
	Areas    []area  `json:"areas"`
}
