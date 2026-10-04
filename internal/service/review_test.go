package service

import (
	"context"
	"errors"

	"pricefollower.local/internal/claude"
	"pricefollower.local/internal/model"
)

func (f *fakeClaude) Review(context.Context, string, claude.ReviewInput) (model.AIReviewContent, error) {
	return model.AIReviewContent{}, errors.New("not used by settings tests")
}
