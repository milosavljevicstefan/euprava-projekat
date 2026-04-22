package main

import "context"

type OpenDataVaspitacView struct {
	VaspitacEmail string `json:"vaspitac_email"`
	VrticID       string `json:"vrtic_id"`
	VrticNaziv    string `json:"vrtic_naziv"`
}

func getOpenDataEducators(ctx context.Context) ([]OpenDataVaspitacView, error) {
	items, err := listAssignments(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]OpenDataVaspitacView, 0, len(items))
	for _, item := range items {
		result = append(result, OpenDataVaspitacView{
			VaspitacEmail: item.VaspitacEmail,
			VrticID:       item.VrticID.Hex(),
			VrticNaziv:    item.VrticNaziv,
		})
	}
	return result, nil
}
