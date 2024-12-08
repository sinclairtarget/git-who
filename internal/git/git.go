/*
* Wraps access to data needed from Git.
*
* We invoke Git directly as a subprocess and parse the output rather than using
* git2go/libgit2.
*/
package git

// Whether we rank authors by commit, lines, or files.
type TallyMode int

const (
    CommitMode TallyMode = iota
    LinesMode 
    FilesMode
)


