/**
 *
 */
package static

import "testing"

func TestUseStatik(t *testing.T) {
	UseStatik("/a.json")
	UseStatik("/dir1/dir2/b.json")
}
