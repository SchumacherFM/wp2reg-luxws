package luxwsclient

type ItemJSON struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Id    string `json:"id"`
}

type NavigationJSON struct {
	Type  string `json:"type"`
	Items []struct {
		Items []struct {
			Items []struct {
				Items    []*ItemJSON `json:"items"`
				Id       string      `json:"id"`
				Name     string      `json:"name"`
				ReadOnly bool        `json:"readOnly"`
			} `json:"items"`
			Id       string `json:"id"`
			Name     string `json:"name"`
			ReadOnly bool   `json:"readOnly"`
		} `json:"items"`
		Id       string `json:"id"`
		Name     string `json:"name"`
		ReadOnly bool   `json:"readOnly"`
	} `json:"items"`
	Id string `json:"id"`
}

type ContentJSON struct {
	Type  string `json:"type"`
	Items []struct {
		Name  string `json:"name"`
		Items []struct {
			Name  string      `json:"name"`
			Value string      `json:"value,omitempty"`
			Id    string      `json:"id"`
			Items []*ItemJSON `json:"items,omitempty"`
		} `json:"items"`
		Id string `json:"id"`
	} `json:"items"`
	Name string `json:"name"`
}
