package earslint

func LintEars(text string, catalog Catalog, options *Options) LintResult {
	opts := withDefaults(options)
	return lintOne("", text, catalog, opts)
}

func LintEarsBatch(items [][2]string, catalog Catalog, options *Options) []LintResult {
	opts := withDefaults(options)
	results := make([]LintResult, 0, len(items))
	for _, item := range items {
		results = append(results, lintOne(item[0], item[1], catalog, opts))
	}
	return results
}

func withDefaults(options *Options) Options {
	if options == nil {
		return Options{
			Mode:       ModeStrict,
			VagueTerms: []string{"appropriate", "sufficient", "as needed"},
		}
	}
	out := *options
	if out.Mode == "" {
		out.Mode = ModeStrict
	}
	if len(out.VagueTerms) == 0 {
		out.VagueTerms = []string{"appropriate", "sufficient", "as needed"}
	}
	return out
}
