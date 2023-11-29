package infra

type Infra struct {
	Compute Compute `json:"compute"`
	Network Network `json:"network"`
	GPU     GPU     `json:"gpu"`
}
