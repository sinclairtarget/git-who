/*
* Implements all the subcommands available via the CLI.
 */
package subcommands

import "time"

var progStart time.Time

func init() {
	progStart = time.Now()
}
