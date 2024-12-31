// https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang
package flagutils

import "fmt"

type SliceFlag []string

func (s *SliceFlag) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *SliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}
