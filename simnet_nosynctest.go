//go:build !goexperiment.synctest

package gosimnet

const globalUseSynctest bool = false

func synctestWait_LetAllOtherGoroFinish() {}

func bubbleOrNot(f func()) {
	f()
}
