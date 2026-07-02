// Command daemon is the privileged enforcer described in
// docs/ENFORCEMENT.md §4.3. Stage 0 is a placeholder so the module compiles
// and the binary builds; the lock state machine, heartbeat server, and
// per-OS backends land starting in Stage 3.
package main

import (
	"fmt"

	_ "github.com/cnqso/sludge-exploder/shared"
)

func main() {
	fmt.Println("sludge-exploder daemon: not yet implemented (Stage 3)")
}
