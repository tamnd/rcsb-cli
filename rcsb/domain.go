package rcsb

import (
	"context"
	"fmt"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "rcsb",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "rcsb",
			Short:  "A command line for the RCSB Protein Data Bank.",
			Long: `A command line for the RCSB Protein Data Bank.

rcsb reads protein and macromolecular structures from data.rcsb.org,
the primary archive of 3D structural data for biological molecules.
No API key required. 255,000+ experimentally determined structures.`,
			Site: "https://www.rcsb.org",
			Repo: "https://github.com/tamnd/rcsb-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "search", Group: "read", List: true,
		Summary: "Search structures by keyword (--limit, --start)",
		Args:    []kit.Arg{{Name: "query", Help: "text search query"}}}, searchStructures)

	kit.Handle(app, kit.OpMeta{Name: "entry", Group: "read", Single: true,
		Summary: "Get a single structure by PDB ID (e.g. 4HHB)",
		Args:    []kit.Arg{{Name: "id", Help: "4-letter PDB ID"}}}, getEntry)

	kit.Handle(app, kit.OpMeta{Name: "structures", Group: "read", List: true,
		Summary: "List all current PDB entry IDs (--limit)"}, listStructures)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

type searchInput struct {
	Query  string  `kit:"arg"          help:"text search query"`
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Start  int     `kit:"flag"         help:"offset for pagination"`
	Client *Client `kit:"inject"`
}

type entryInput struct {
	ID     string  `kit:"arg"    help:"4-letter PDB ID"`
	Client *Client `kit:"inject"`
}

type structuresInput struct {
	Limit  int     `kit:"flag,inherit" help:"max results (0 = all)"`
	Client *Client `kit:"inject"`
}

func searchStructures(ctx context.Context, in searchInput, emit func(*SearchResult) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	results, _, err := in.Client.SearchStructures(ctx, in.Query, limit, in.Start)
	if err != nil {
		return err
	}
	for i := range results {
		if err := emit(&results[i]); err != nil {
			return err
		}
	}
	return nil
}

func getEntry(ctx context.Context, in entryInput, emit func(*Structure) error) error {
	s, err := in.Client.GetEntry(ctx, in.ID)
	if err != nil {
		return err
	}
	return emit(s)
}

func listStructures(ctx context.Context, in structuresInput, emit func(*SearchResult) error) error {
	ids, err := in.Client.ListEntryIDs(ctx)
	if err != nil {
		return err
	}
	limit := in.Limit
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}
	for _, id := range ids {
		if err := emit(&SearchResult{ID: id}); err != nil {
			return err
		}
	}
	return nil
}

func (Domain) Classify(input string) (string, string, error) {
	upper := strings.ToUpper(strings.TrimSpace(input))
	if len(upper) == 4 {
		return "entry", upper, nil
	}
	return "", "", errs.Usage("rcsb IDs are 4 characters (e.g. 4HHB), got %q", input)
}

func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "entry":
		return fmt.Sprintf("https://www.rcsb.org/structure/%s", strings.ToUpper(id)), nil
	default:
		return "", errs.Usage("rcsb has no resource type %q", t)
	}
}
