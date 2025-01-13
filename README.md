# git-who
 ![Vanity screenshot](./screenshots/vanity.png)

`git-who` is a command-line tool for answering that age-old question:

> _Who wrote this code?!_

Unlike `git blame`, which can tell you who wrote a _line_ of code, `git-who`
can help you identify the people responsible for entire components or
subsystems in a codebase. You can think of `git-who` sort of like `git-blame`
but for file trees rather than individual files.

## Installation
TBD.

## Usage
_(In the following examples, `git-who` is invoked as `git who`, which requires
setting up a Git alias. See the [Git Alias](#git-alias) section below.)_

`git who` has three subcommands. Each subcommand gives you a different view of
authorship in your Git repository.

### The `table` Subcommand
The `table` subcommand is the default subcommand. You can invoke it explicitly
as `git who table` or implicitly just as `git who`.

The `table` subcommand prints a table summarizing the contributions of every
author who has made commits in the repository:

```
~/repos/cpython$ git who
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  2 mon. ago    11213│
│Victor Stinner                    1 week ago     7193│
│Fred Drake                        13 yr. ago     5465│
│Georg Brandl                      1 year ago     5294│
│Benjamin Peterson                 4 mon. ago     4724│
│Raymond Hettinger                 1 month ago    4235│
│Serhiy Storchaka                  1 day ago      3366│
│Antoine Pitrou                    10 mon. ago    3180│
│Jack Jansen                       18 yr. ago     2978│
│Martin v. Löwis                   9 yr. ago      2690│
│...3026 more...                                      │
└─────────────────────────────────────────────────────┘
```

You can specify a path to filter the results to only commits that
touched files under the given path:
```
~/repos/cpython$ git who Tools/
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  8 mon. ago      820│
│Barry Warsaw                      1 year ago      279│
│Martin v. Löwis                   9 yr. ago       242│
│Victor Stinner                    1 month ago     235│
│Steve Dower                       1 month ago     228│
│Jeremy Hylton                     19 yr. ago      178│
│Mark Shannon                      4 hr. ago       131│
│Serhiy Storchaka                  2 mon. ago      118│
│Erlend E. Aasland                 1 week ago      117│
│Christian Heimes                  2 yr. ago       114│
│...267 more...                                       │
└─────────────────────────────────────────────────────┘
```

You can also specify a branch name, tag name, or any "commit-ish" to
filter the results to commits reachable from the specified commit:
```
~/repos/cpython$ git who v3.7.1
┌─────────────────────────────────────────────────────┐
│Author                            Last Edit   Commits│
├─────────────────────────────────────────────────────┤
│Guido van Rossum                  6 yr. ago     10986│
│Fred Drake                        13 yr. ago     5465│
│Georg Brandl                      8 yr. ago      5291│
│Benjamin Peterson                 6 yr. ago      4599│
│Victor Stinner                    6 yr. ago      4462│
│Raymond Hettinger                 6 yr. ago      3667│
│Antoine Pitrou                    6 yr. ago      3149│
│Jack Jansen                       18 yr. ago     2978│
│Martin v. Löwis                   9 yr. ago      2690│
│Tim Peters                        10 yr. ago     2489│
│...550 more...                                       │
└─────────────────────────────────────────────────────┘
```

Just like with `git` itself, when there is ambiguity between a path name
and a commit-ish, you can use `--` to clarfiy the distinction. The
following command will show you contributions to the file or directory
called `foo` even if there is also a branch called `foo` in your repository:
```
$ git who -- foo
```

#### Options
The `-m`, `-l`, and `-f` flags allow you to sort the table by different
metrics.

The `-m` flag sorts the table by the "Last Edit" column, showing who
edited the repository most recently.

The `-l` flag sorts the table by number of lines modified, adding some more
columns:

```
$ git who -l
┌──────────────────────────────────────────────────────────────────────────────┐
│Author                           Last Edit   Commits   Files       Lines (+/-)│
├──────────────────────────────────────────────────────────────────────────────┤
│Jiang Xin                        8 mon. ago      331     471  253283 /  226023│
│Junio C Hamano                   19 hr. ago     7993    2492  247398 /  105550│
│Peter Krefting                   3 mon. ago      122     564  160447 /  135430│
│Tran Ngoc Quan                   2 yr. ago        84     296  148753 /  126581│
│Alexander Shopov                 1 week ago       82     298  143828 /  114936│
│Jordi Mas                        2 mon. ago       59     222  110952 /   95889│
│Jean-Noël Avila                  1 month ago     126     383   96875 /   90328│
│Ralf Thielow                     2 mon. ago      166     470   88881 /   64093│
│Dimitriy Ryazantcev              1 year ago       30     279   84741 /   62953│
│Jeff King                        3 days ago     4421    2056   86235 /   47464│
│...2243 more...                                                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

The `-f` flag sorts the table by the number of files modified.

There is also a `--csv` option that outputs the table as a CSV file to stdout.

Run `git-who table --help` to see additional options for the `table` subcommand.

### The `tree` Subcommand
The `tree` subcommand prints out a file tree showing files in the working tree
just like [tree](https://en.wikipedia.org/wiki/Tree_(command)). Each node in the
file tree is annotated with information showing which author contributed the most
to files at or under that path.

Here is an example showing contributions to the Python parser. By default,
contributions will be measured by number of commits:
```
~/repos/cpython$ git who tree Parser/
Parser/.........................Guido van Rossum (182)
├── lexer/......................Pablo Galindo Salgado (5)
│   ├── buffer.c................Lysandros Nikolaou (1)
│   ├── buffer.h................Lysandros Nikolaou (1)
│   ├── lexer.c
│   ├── lexer.h.................Lysandros Nikolaou (1)
│   ├── state.c
│   └── state.h
├── tokenizer/..................Filipe Laíns (1)
│   ├── file_tokenizer.c
│   ├── helpers.c...............Lysandros Nikolaou (1)
│   ├── helpers.h...............Lysandros Nikolaou (1)
│   ├── readline_tokenizer.c....Lysandros Nikolaou (1)
│   ├── string_tokenizer.c......Lysandros Nikolaou (1)
│   ├── tokenizer.h.............Lysandros Nikolaou (1)
│   └── utf8_tokenizer.c........Lysandros Nikolaou (1)
├── Python.asdl.................Benjamin Peterson (14)
├── action_helpers.c............Pablo Galindo Salgado (6)
├── asdl.py.....................Benjamin Peterson (7)
├── asdl_c.py...................Benjamin Peterson (42)
├── myreadline.c
├── parser.c....................Pablo Galindo Salgado (34)
├── peg_api.c...................Lysandros Nikolaou (2)
├── pegen.c.....................Pablo Galindo (33)
├── pegen.h.....................Pablo Galindo Salgado (13)
├── pegen_errors.c..............Pablo Galindo Salgado (16)
├── string_parser.c.............Victor Stinner (10)
├── string_parser.h.............Pablo Galindo Salgado (1)
└── token.c.....................Pablo Galindo Salgado (2)
```

You may notice that some files, like `lexer.c`, are not annotated.
If a file is not annotated, that is because the author who has
most contributed to that file is the same as the author who
has most contributed to the directory containing the file. This is
done to minimize visual noise.

You can force `git-who tree` to annotate every file using the `-a`
flag (for "all"). This flag also prints all file paths that
were discovered while walking the commit history, including those no
longer in the working tree:

```
~/repos/cpython$ git who tree -a Parser/
Parser/.........................Guido van Rossum (182)
├── lexer/......................Pablo Galindo Salgado (5)
│   ├── buffer.c................Lysandros Nikolaou (1)
│   ├── buffer.h................Lysandros Nikolaou (1)
│   ├── lexer.c.................Pablo Galindo Salgado (4)
│   ├── lexer.h.................Lysandros Nikolaou (1)
│   ├── state.c.................Pablo Galindo Salgado (2)
│   └── state.h.................Pablo Galindo Salgado (1)
├── pegen/......................Pablo Galindo (30)
│   ├── parse.c.................Pablo Galindo (16)
│   ├── parse_string.c..........Pablo Galindo (7)
│   ├── parse_string.h..........Pablo Galindo (2)
│   ├── peg_api.c...............Pablo Galindo (3)
│   ├── pegen.c.................Pablo Galindo (17)
│   └── pegen.h.................Pablo Galindo (9)
├── pgen/.......................Pablo Galindo (8)
│   ├── __init__.py.............Pablo Galindo (2)
│   ├── __main__.py.............Pablo Galindo (5)
│   ├── automata.py.............Pablo Galindo (4)
│   ├── grammar.py..............Pablo Galindo (5)
│   ├── keywordgen.py...........Pablo Galindo (3)
│   ├── metaparser.py...........Pablo Galindo (2)
│   ├── pgen.py.................Pablo Galindo (5)
│   └── token.py................Pablo Galindo (4)
├── tokenizer/..................Filipe Laíns (1)
│   ├── file_tokenizer.c........Filipe Laíns (1)
│   ├── helpers.c...............Lysandros Nikolaou (1)
│   ├── helpers.h...............Lysandros Nikolaou (1)
│   ├── readline_tokenizer.c....Lysandros Nikolaou (1)
│   ├── string_tokenizer.c......Lysandros Nikolaou (1)
│   ├── tokenizer.h.............Lysandros Nikolaou (1)
│   └── utf8_tokenizer.c........Lysandros Nikolaou (1)
├── .cvsignore..................Martin v. Löwis (1)
├── Makefile.in.................Guido van Rossum (10)
├── Python.asdl.................Benjamin Peterson (14)
├── acceler.c...................Guido van Rossum (17)
├── action_helpers.c............Pablo Galindo Salgado (6)
├── asdl.py.....................Benjamin Peterson (7)
├── asdl_c.py...................Benjamin Peterson (42)
├── assert.h....................Guido van Rossum (11)
├── bitset.c....................Guido van Rossum (12)
├── firstsets.c.................Guido van Rossum (13)
├── grammar.c...................Guido van Rossum (20)
...
```
(_The above output continues but has been elided for the purposes 
of this README._)

Note that, whether or not the `-a` flag is used, commits that
edited files
not in the working tree will still count toward the total displayed
next to ancestor directories of that file. In the above two examples,
Guido van Rossum is shown as the overall highest committer to the
`Parser/` directory, though it takes listing the entire
tree with the `-a` flag to see that most of his commits were to
files that have since been moved or deleted.

The `tree` subcommand, like the `table` subcommand, supports the `-l`, `-f`,
and `-m` flags. The `-l` flag will annotate each file tree node with the
author who has added or removed the most lines at that path:

```
~/repos/cpython$ git who tree -l Parser/
Parser/.........................Pablo Galindo (72917 / 47102)
├── lexer/......................Lysandros Nikolaou (1668 / 0)
│   ├── buffer.c
│   ├── buffer.h
│   ├── lexer.c
│   ├── lexer.h
│   ├── state.c
│   └── state.h.................Pablo Galindo Salgado (1 / 0)
├── tokenizer/..................Lysandros Nikolaou (1391 / 0)
│   ├── file_tokenizer.c
│   ├── helpers.c
│   ├── helpers.h
│   ├── readline_tokenizer.c
│   ├── string_tokenizer.c
│   ├── tokenizer.h
│   └── utf8_tokenizer.c
├── Python.asdl.................Benjamin Peterson (120 / 122)
├── action_helpers.c
├── asdl.py.....................Eli Bendersky (276 / 331)
├── asdl_c.py...................Victor Stinner (634 / 496)
├── myreadline.c................Guido van Rossum (365 / 226)
├── parser.c
├── peg_api.c...................Victor Stinner (5 / 46)
├── pegen.c
├── pegen.h
├── pegen_errors.c
├── string_parser.c
├── string_parser.h
└── token.c.....................Serhiy Storchaka (233 / 0)
```

The `-f` flag will pick authors based on files touched and the `-m` flag will
pick an author based on last modification time.

You can limit the depth of the tree printed by using the `-d` flag. The depth
is measured from the current working directory. 

Run `git who tree --help` to see all options available for the `tree` subcommand.

### The `hist` Subcommand
TODO.

## Git Alias
You can invoke `git-who` as `git who` by setting up an alias in your global Git
config:

```
[alias]
    who = "!git-who"
```

See [here](https://git-scm.com/book/en/v2/Git-Basics-Git-Aliases) for more
information about Git aliases.

## DEVELOPMENT
### Test Repository Submodule
Some of the automated tests for `git-who` need to run against a Git repository.
A test repository is attached to this repository as a submodule.

If you want to run the automated tests, you will first need to set up the
submodule:

```
$ git submodule update --init
```
