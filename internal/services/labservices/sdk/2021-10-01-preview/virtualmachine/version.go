package virtualmachine

import "fmt"

const defaultApiVersion = "2021-10-01-preview"

func userAgent() string {
	return fmt.Sprintf("pandora/virtualmachine/%s", defaultApiVersion)
}
