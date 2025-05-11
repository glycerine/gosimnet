//go:build goexperiment.synctest

package gosimnet

import (
	"fmt"
	"testing/synctest"
)

const globalUseSynctest bool = true

func init() {
	fmt.Printf("globalUseSynctest = %v\n", globalUseSynctest)
}

func bubbleOrNot(f func()) {
	synctest.Run(f)
}

func synctestWait_LetAllOtherGoroFinish() {
	synctest.Wait()
}
