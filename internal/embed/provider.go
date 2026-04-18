package embed

import (
	"fmt"
)

type Provider interface {
	Embed(text string) ([]float32, error)
	EmbedBatch(texts []string) ([][]float32, error)
	Dimensions() int
	Name() string
}

type NoneProvider struct{}

func (p *NoneProvider) Embed(text string) ([]float32, error) {
	return nil, fmt.Errorf("embedding not configured (provider: none)")
}

func (p *NoneProvider) EmbedBatch(texts []string) ([][]float32, error) {
	return nil, fmt.Errorf("embedding not configured (provider: none)")
}

func (p *NoneProvider) Dimensions() int { return 0 }
func (p *NoneProvider) Name() string    { return "none" }
