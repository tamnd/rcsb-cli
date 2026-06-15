package rcsb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Host = "data.rcsb.org"
const baseURL = "https://data.rcsb.org/rest/v1"
const searchURL = "https://search.rcsb.org/rcsbsearch/v2/query"

type Config struct {
	BaseURL   string
	SearchURL string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   baseURL,
		SearchURL: searchURL,
		Rate:      300 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
		UserAgent: "rcsb-cli/0.1.0 (github.com/tamnd/rcsb-cli)",
	}
}

type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) wait() {
	if c.cfg.Rate > 0 {
		if since := time.Since(c.last); since < c.cfg.Rate {
			time.Sleep(c.cfg.Rate - since)
		}
	}
	c.last = time.Now()
}

func (c *Client) get(ctx context.Context, rawURL string, out any) error {
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			d := time.Duration(attempt) * 500 * time.Millisecond
			if d > 5*time.Second {
				d = 5 * time.Second
			}
			time.Sleep(d)
		}
		c.wait()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", c.cfg.UserAgent)
		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < c.cfg.Retries {
				continue
			}
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("not found")
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			if attempt < c.cfg.Retries {
				continue
			}
			n := len(body)
			if n > 200 {
				n = 200
			}
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body)[:n])
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return fmt.Errorf("all retries exhausted")
}

// --- wire types ---

type wireEntry struct {
	Entry struct {
		ID string `json:"id"`
	} `json:"entry"`
	Struct struct {
		Title string `json:"title"`
	} `json:"struct"`
	RcsbEntryInfo struct {
		ExperimentalMethod        string    `json:"experimental_method"`
		ResolutionCombined        []float64 `json:"resolution_combined"`
		PolymerEntityCountProtein int       `json:"polymer_entity_count_protein"`
		MolecularWeight           float64   `json:"molecular_weight"`
		DepositedAtomCount        int       `json:"deposited_atom_count"`
		PolymerEntityCount        int       `json:"polymer_entity_count"`
		NonpolymerEntityCount     int       `json:"nonpolymer_entity_count"`
	} `json:"rcsb_entry_info"`
	Exptl []struct {
		Method string `json:"method"`
	} `json:"exptl"`
	RcsbAccessionInfo struct {
		InitialReleaseDate string `json:"initial_release_date"`
		DepositDate        string `json:"deposit_date"`
	} `json:"rcsb_accession_info"`
	RcsbPrimaryCitation struct {
		RcsbAuthors   []string `json:"rcsb_authors"`
		Title         string   `json:"title"`
		JournalAbbrev string   `json:"journal_abbrev"`
		Year          int      `json:"year"`
		DOI           string   `json:"pdbx_database_id_DOI"`
	} `json:"rcsb_primary_citation"`
	StructKeywords struct {
		PdbxKeywords string `json:"pdbx_keywords"`
		Text         string `json:"text"`
	} `json:"struct_keywords"`
}

type wireSearchResult struct {
	TotalCount int `json:"total_count"`
	ResultSet  []struct {
		Identifier string  `json:"identifier"`
		Score      float64 `json:"score"`
	} `json:"result_set"`
}

// --- public output types ---

type Structure struct {
	ID              string   `json:"id"                 kit:"id"`
	Title           string   `json:"title"`
	Method          string   `json:"method"`
	Resolution      float64  `json:"resolution,omitempty"`
	MolecularWeight float64  `json:"molecular_weight,omitempty"`
	ProteinChains   int      `json:"protein_chains,omitempty"`
	AtomCount       int      `json:"atom_count,omitempty"`
	Keywords        string   `json:"keywords,omitempty"`
	Authors         []string `json:"authors,omitempty"`
	CitationTitle   string   `json:"citation_title,omitempty"`
	Journal         string   `json:"journal,omitempty"`
	Year            int      `json:"year,omitempty"`
	DOI             string   `json:"doi,omitempty"`
	DepositDate     string   `json:"deposit_date,omitempty"`
	ReleaseDate     string   `json:"release_date,omitempty"`
}

type SearchResult struct {
	ID    string  `json:"id"    kit:"id"`
	Score float64 `json:"score"`
}

func toStructure(w wireEntry) *Structure {
	var method string
	if len(w.Exptl) > 0 {
		method = w.Exptl[0].Method
	}
	var res float64
	if len(w.RcsbEntryInfo.ResolutionCombined) > 0 {
		res = w.RcsbEntryInfo.ResolutionCombined[0]
	}
	releaseDate := ""
	if len(w.RcsbAccessionInfo.InitialReleaseDate) >= 10 {
		releaseDate = w.RcsbAccessionInfo.InitialReleaseDate[:10]
	}
	depositDate := ""
	if len(w.RcsbAccessionInfo.DepositDate) >= 10 {
		depositDate = w.RcsbAccessionInfo.DepositDate[:10]
	}
	return &Structure{
		ID:              strings.ToUpper(w.Entry.ID),
		Title:           w.Struct.Title,
		Method:          method,
		Resolution:      res,
		MolecularWeight: w.RcsbEntryInfo.MolecularWeight,
		ProteinChains:   w.RcsbEntryInfo.PolymerEntityCountProtein,
		AtomCount:       w.RcsbEntryInfo.DepositedAtomCount,
		Keywords:        w.StructKeywords.Text,
		Authors:         w.RcsbPrimaryCitation.RcsbAuthors,
		CitationTitle:   w.RcsbPrimaryCitation.Title,
		Journal:         w.RcsbPrimaryCitation.JournalAbbrev,
		Year:            w.RcsbPrimaryCitation.Year,
		DOI:             w.RcsbPrimaryCitation.DOI,
		DepositDate:     depositDate,
		ReleaseDate:     releaseDate,
	}
}

// GetEntry fetches a single PDB entry by 4-letter ID.
func (c *Client) GetEntry(ctx context.Context, pdbID string) (*Structure, error) {
	var w wireEntry
	u := fmt.Sprintf("%s/core/entry/%s", c.cfg.BaseURL, strings.ToUpper(pdbID))
	if err := c.get(ctx, u, &w); err != nil {
		return nil, err
	}
	return toStructure(w), nil
}

// SearchStructures performs a text search for structures.
func (c *Client) SearchStructures(ctx context.Context, query string, limit, start int) ([]SearchResult, int, error) {
	qJSON := fmt.Sprintf(`{"query":{"type":"terminal","service":"text","parameters":{"value":%s}},"return_type":"entry","request_options":{"paginate":{"start":%d,"rows":%d}}}`,
		jsonStr(query), start, limit)
	u := c.cfg.SearchURL + "?json=" + url.QueryEscape(qJSON)
	var w wireSearchResult
	if err := c.get(ctx, u, &w); err != nil {
		return nil, 0, err
	}
	out := make([]SearchResult, len(w.ResultSet))
	for i, r := range w.ResultSet {
		out[i] = SearchResult{ID: r.Identifier, Score: r.Score}
	}
	return out, w.TotalCount, nil
}

// ListEntryIDs returns all current PDB entry IDs.
func (c *Client) ListEntryIDs(ctx context.Context) ([]string, error) {
	u := c.cfg.BaseURL + "/holdings/current/entry_ids"
	var ids []string
	if err := c.get(ctx, u, &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
