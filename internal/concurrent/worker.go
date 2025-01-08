package concurrent

import (
	"fmt"

	"github.com/sinclairtarget/git-who/internal/git"
)

// Writes revs to git log stdin
func writeWorker[T any](
	workload workload[T],
	subprocess *git.Subprocess,
) (err error) {
	defer func() {
		if err != nil {
			workload.writeError <- err
		}

		close(workload.writeError)
	}()

	w, closer := subprocess.StdinWriter()
	for _, rev := range workload.revs {
		fmt.Fprintln(w, rev)
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	err = closer()
	if err != nil {
		return err
	}

	return nil
}

func tallyWorker[T any, R any](
	workload workload[T],
	subprocess *git.Subprocess,
	whop Whoperation[T, R],
) (err error) {
	defer func() {
		if err != nil {
			workload.tallyError <- err
		}

		close(workload.tallyError)
		close(workload.result)
	}()

	lines := subprocess.StdoutLines()
	commits := git.ParseCommits(lines)

	result, err := whop.Apply(commits)
	if err != nil {
		return err
	}

	workload.result <- result
	return nil
}
