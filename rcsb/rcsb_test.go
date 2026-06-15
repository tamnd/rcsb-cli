package rcsb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.SearchURL = srv.URL + "/search"
	cfg.Rate = 0
	cfg.Retries = 0
	return srv, NewClient(cfg)
}

func TestGetEntry(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/core/entry/4HHB", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(wireEntry{
			Entry:  struct{ ID string `json:"id"` }{ID: "4HHB"},
			Struct: struct{ Title string `json:"title"` }{Title: "HUMAN DEOXYHAEMOGLOBIN"},
			Exptl:  []struct{ Method string `json:"method"` }{{Method: "X-RAY DIFFRACTION"}},
			RcsbEntryInfo: struct {
				ExperimentalMethod        string    `json:"experimental_method"`
				ResolutionCombined        []float64 `json:"resolution_combined"`
				PolymerEntityCountProtein int       `json:"polymer_entity_count_protein"`
				MolecularWeight           float64   `json:"molecular_weight"`
				DepositedAtomCount        int       `json:"deposited_atom_count"`
				PolymerEntityCount        int       `json:"polymer_entity_count"`
				NonpolymerEntityCount     int       `json:"nonpolymer_entity_count"`
			}{
				ExperimentalMethod:        "X-ray",
				ResolutionCombined:        []float64{1.74},
				PolymerEntityCountProtein: 2,
				MolecularWeight:           64.74,
				DepositedAtomCount:        4779,
			},
			RcsbAccessionInfo: struct {
				InitialReleaseDate string `json:"initial_release_date"`
				DepositDate        string `json:"deposit_date"`
			}{
				InitialReleaseDate: "1984-07-17T00:00:00.000+00:00",
				DepositDate:        "1984-03-07T00:00:00.000+00:00",
			},
		})
	})
	_, client := testServer(t, mux)
	s, err := client.GetEntry(context.Background(), "4HHB")
	if err != nil {
		t.Fatal(err)
	}
	if s.ID != "4HHB" {
		t.Errorf("ID = %q, want 4HHB", s.ID)
	}
	if s.Method != "X-RAY DIFFRACTION" {
		t.Errorf("Method = %q, want X-RAY DIFFRACTION", s.Method)
	}
	if s.Resolution != 1.74 {
		t.Errorf("Resolution = %v, want 1.74", s.Resolution)
	}
	if s.MolecularWeight != 64.74 {
		t.Errorf("MolecularWeight = %v, want 64.74", s.MolecularWeight)
	}
}

func TestSearchStructures(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(wireSearchResult{
			TotalCount: 1234,
			ResultSet: []struct {
				Identifier string  `json:"identifier"`
				Score      float64 `json:"score"`
			}{
				{Identifier: "4HHB", Score: 1.0},
				{Identifier: "1MBN", Score: 0.9},
			},
		})
	})
	_, client := testServer(t, mux)
	results, total, err := client.SearchStructures(context.Background(), "hemoglobin", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1234 {
		t.Errorf("total = %d, want 1234", total)
	}
	if len(results) != 2 {
		t.Errorf("len = %d, want 2", len(results))
	}
	if results[0].ID != "4HHB" {
		t.Errorf("results[0].ID = %q, want 4HHB", results[0].ID)
	}
}

func TestListEntryIDs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/holdings/current/entry_ids", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]string{"4HHB", "1MBN", "2HHB"})
	})
	_, client := testServer(t, mux)
	ids, err := client.ListEntryIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 3 {
		t.Errorf("len = %d, want 3", len(ids))
	}
	if ids[0] != "4HHB" {
		t.Errorf("ids[0] = %q, want 4HHB", ids[0])
	}
}

func TestGetEntryNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/core/entry/XXXX", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	_, client := testServer(t, mux)
	_, err := client.GetEntry(context.Background(), "XXXX")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
