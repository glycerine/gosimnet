//go:build goexperiment.synctest

package gosimnet

import (
	"testing/synctest"
)

func bubbleOrNot(f func()) {
	synctest.Run(f)
}

const globalUseSynctest bool = true

func synctestWait_LetAllOtherGoroFinish() {
	synctest.Wait()
}
