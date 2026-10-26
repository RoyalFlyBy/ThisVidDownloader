package utils

// AppArgs holds the parsed command-line flags. Every getter clamps or defaults
// its value so the rest of the app never has to.
type AppArgs struct {
	Video    *string
	List     *string
	Member   *string
	Playlist *string
	Output   *string
	Proxy    *string
	Cookies  *string
	Debug    *bool
	Version  *bool
	PrefixID *bool
	Threads  *int
	Amount   *int
	Offset   *int
	Quality  *int
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func boolVal(p *bool) bool {
	return p != nil && *p
}

func intVal(p *int, def int) int {
	if p == nil {
		return def
	}
	return *p
}

func (args *AppArgs) DoVideo() bool    { return len(str(args.Video)) > 0 }
func (args *AppArgs) DoList() bool     { return len(str(args.List)) > 0 }
func (args *AppArgs) DoMember() bool   { return len(str(args.Member)) > 0 }
func (args *AppArgs) DoPlaylist() bool { return len(str(args.Playlist)) > 0 }

func (args *AppArgs) GetVideo() string    { return str(args.Video) }
func (args *AppArgs) GetList() string     { return str(args.List) }
func (args *AppArgs) GetMember() string   { return str(args.Member) }
func (args *AppArgs) GetPlaylist() string { return str(args.Playlist) }
func (args *AppArgs) GetProxy() string    { return str(args.Proxy) }
func (args *AppArgs) GetCookies() string  { return str(args.Cookies) }
func (args *AppArgs) GetPrefixID() bool   { return boolVal(args.PrefixID) }
func (args *AppArgs) GetDebug() bool      { return boolVal(args.Debug) }
func (args *AppArgs) GetVersion() bool    { return boolVal(args.Version) }

func (args *AppArgs) GetOutput() string {
	if out := str(args.Output); out != "" {
		return out
	}
	return "."
}

func (args *AppArgs) GetThreads() int {
	threads := intVal(args.Threads, 4)
	switch {
	case threads < 1:
		return 1
	case threads > 12:
		return 12
	default:
		return threads
	}
}

func (args *AppArgs) GetAmount() int {
	if amount := intVal(args.Amount, 20); amount > 0 {
		return amount
	}
	return 1
}

func (args *AppArgs) GetOffset() int {
	if offset := intVal(args.Offset, 0); offset > 0 {
		return offset
	}
	return 0
}

// GetQuality returns -1 for the best stream, or an index into the player's
// stream list.
func (args *AppArgs) GetQuality() int {
	if quality := intVal(args.Quality, -1); quality >= 0 {
		return quality
	}
	return -1
}
