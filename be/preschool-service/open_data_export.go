package main

import "context"

type OpenDataZahtevView struct {
	ID           string `json:"id"`
	ImeRoditelja string `json:"ime_roditelja"`
	Status       string `json:"status"`
}

func getOpenDataRequests(ctx context.Context) ([]OpenDataZahtevView, error) {
	items, err := getAllRequests(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]OpenDataZahtevView, 0, len(items))
	for _, item := range items {
		result = append(result, OpenDataZahtevView{
			ID:           item.ID.Hex(),
			ImeRoditelja: item.ImeRoditelja,
			Status:       canonicalRequestStatus(item.Status),
		})
	}

	return result, nil
}
