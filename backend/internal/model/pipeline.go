// Package model holds the domain types shared by every layer. It imports nothing from internal/.
package model

import "time"

// Product is the product line a pipeline belongs to.
type Product string

const (
	ProductFlashArray Product = "flasharray"
	ProductFlashBlade Product = "flashblade"
	ProductHyperscale Product = "hyperscale"
)

// Pipeline is the full shape we store and send back.
type Pipeline struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Product   Product   `json:"product"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreatePipelineInput is what a client may send on POST /pipelines.
// No ID or CreatedAt: the server sets those.
type CreatePipelineInput struct {
	Name    string  `json:"name"`
	Product Product `json:"product"`
}
