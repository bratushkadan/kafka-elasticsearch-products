package pkg

type Product struct {
	Id          string `json:"string"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
	Seller      string `json:"seller"`
}
