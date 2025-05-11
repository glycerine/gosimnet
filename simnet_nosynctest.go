//go:build !goexperiment.synctest

package gosimnet

import (
	"fmt"
)

const globalUseSynctest bool = false

func init() {
	fmt.Printf("globalUseSynctest = %v\n", globalUseSynctest)
}

func synctestWait_LetAllOtherGoroFinish() {}

func bubbleOrNot(f func()) {
	f()
}
