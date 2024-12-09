package git

// Whether we rank authors by commit, lines, or files.
type TallyMode int

const (
    CommitMode TallyMode = iota
    LinesMode
    FilesMode
)

type Tally struct {
	Path         string
	AuthorName   string
	AuthorEmail  string
	Commits      int
	LinesAdded   int
	LinesRemoved int
	Files        int
}
